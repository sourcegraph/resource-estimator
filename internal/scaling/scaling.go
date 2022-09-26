package scaling

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type Factor int

const (
	ByEngagedUsers        Factor = iota
	ByAverageRepositories Factor = iota
	ByTotalRepoSize       Factor = iota
	ByLargeMonorepos      Factor = iota
	ByLargestRepoSize     Factor = iota
	ByLargestIndexSize    Factor = iota
	ByUserRepoSumRatio    Factor = iota
)

type Service struct {
	// Value corresponding to the scaling factor type (users, repositories, large monorepos, etc.)
	Value float64 `json:"-"`
	// Optional values indicating that at the specified Value, these service properties are required.
	Replicas                                int       `json:"replicaCount,omitempty"`
	Resources                               Resources `json:"resources,omitempty"`
	Storage                                 float64   `json:"-"`
	PVC                                     string    `json:"storageSize,omitempty"`
	NameInDocker, NameInK8s, PodName, Label string    `json:"-"`
	// ContactSupport, when true, indicates that for the given value support should be contacted.
	ContactSupport bool `json:"-"`
}
type Resources struct {
	Limits   Resource `json:"limits,omitempty"`
	Requests Resource `json:"requests,omitempty"`
}
type Resource struct {
	CPU, MEM, EPH float64 `json:"-"`
	CPUS          string  `json:"cpu,omitempty"`
	MEMS          string  `json:"memory,omitempty"`
	EPHS          string  `json:"ephemeral-storage,omitempty"`
}
type DockerServices struct {
	Version  string                     `json:"version,string"`
	Services map[string]DockerResources `json:"services,omitempty"`
}
type DockerResources struct {
	CPU     string `json:"cpus,string,omitempty"`
	MEM     string `json:"mem_limit,string,omitempty"`
	Storage string `json:"-"`
}
type ResourceRange struct {
	Request, Limit float64
}

func (r ResourceRange) Add(o ResourceRange) ResourceRange {
	return ResourceRange{Request: r.Request + o.Request, Limit: r.Limit + o.Limit}
}

func (r ResourceRange) Sub(o ResourceRange) ResourceRange {
	return ResourceRange{Request: r.Request - o.Request, Limit: r.Limit - o.Limit}
}

func (r ResourceRange) MulScalar(f float64) ResourceRange {
	return ResourceRange{Request: r.Request * f, Limit: r.Limit * f}
}

// resourceRound rounds numbers > 1 to the nearest whole number, and numbers < 1 to the nearest
// quarter (.25, .5, .75, 1).
//
// This ensures 0.25 CPU doesn't get rounded to zero, and ensures 0.517829457364341 CPU gets
// rounded to 0.25.
func resourceRound(f float64) float64 {
	if f > 1 {
		return math.Round(f)
	}
	return math.Round(f*4) / 4
}

func addUnit(f float64, t string) string {
	if f < 1 {
		return fmt.Sprintf("%vM", math.Trunc(f*1000))
	}
	return fmt.Sprintf("%v%v", math.Trunc(f), t)
}

func (r *Service) join(o *Service) {
	r.Value = 0
	r.Label = o.Label
	r.NameInDocker = o.NameInDocker
	r.PodName = o.PodName
	if r.Replicas == 0 {
		r.Replicas = o.Replicas
	}
	if r.Resources.Requests.CPU == 0 && r.Resources.Limits.CPU == 0 {
		r.Resources.Requests.CPU = resourceRound(o.Resources.Requests.CPU)
		r.Resources.Limits.CPU = resourceRound(o.Resources.Limits.CPU)
		r.Resources.Requests.CPUS = strings.ToLower(addUnit(r.Resources.Requests.CPU, ""))
		r.Resources.Limits.CPUS = strings.ToLower(addUnit(r.Resources.Limits.CPU, ""))
	}
	if r.Resources.Requests.MEM == 0 && r.Resources.Limits.MEM == 0 {
		r.Resources.Requests.MEM = resourceRound(o.Resources.Requests.MEM)
		r.Resources.Limits.MEM = resourceRound(o.Resources.Limits.MEM)
		r.Resources.Requests.MEMS = addUnit(r.Resources.Requests.MEM, "G")
		r.Resources.Limits.MEMS = addUnit(r.Resources.Limits.MEM, "G")
	}
	if o.Resources.Limits.EPH > 0 && r.Resources.Requests.EPH == 0 && r.Resources.Limits.EPH == 0 {
		r.Resources.Requests.EPH = resourceRound(o.Resources.Requests.EPH)
		r.Resources.Limits.EPH = resourceRound(o.Resources.Limits.EPH)
		r.Resources.Requests.EPHS = addUnit(resourceRound(math.Floor(r.Resources.Requests.EPH/float64(r.Replicas))), "G")
		r.Resources.Limits.EPHS = addUnit(resourceRound(r.Resources.Limits.EPH/float64(r.Replicas)), "G")
	}
	if o.Storage > 0 {
		r.Storage = resourceRound(o.Storage)
		r.PVC = addUnit(resourceRound(o.Storage), "Gi")
	}
	r.ContactSupport = r.ContactSupport || o.ContactSupport
}

func (d DockerResources) join(o *Service) DockerResources {
	replica := math.Max(float64(o.Replicas), 1)
	d.CPU = strings.ToLower(addUnit(o.Resources.Requests.CPU*replica, ""))
	d.MEM = strings.ToLower(addUnit(o.Resources.Limits.MEM*replica, "g"))
	d.Storage = strings.ToLower(addUnit(o.Storage*float64(o.Replicas), "g"))
	return d
}

type ServiceScale struct {
	ServiceName, ServiceLabel, DockerServiceName, PodName string
	ScalingFactor                                         Factor
	ReferencePoints                                       []Service
}

type Range struct {
	Min, Max float64
}

const (
	// Heuristic which pretends 1 large monorepo == N average repositories.
	MonorepoFactor = 1
)

var (
	UsersRange               = Range{1, 50000}
	RepositoriesRange        = Range{1, 5000000}
	TotalRepoSizeRange       = Range{1, 50000000}
	LargeMonoreposRange      = Range{0, 10}
	LargestRepoSizeRange     = Range{0, 50000000}
	LargestIndexSizeRange    = Range{1, 1000}
	AverageRepositoriesRange = Range{1, 5000000}
	UserRepoSumRatioRange    = Range{1, 5000}
	EngagementRateRange      = Range{5, 100}
)

func init() {
	// Ensure reference points are sorted by ascending value so it is easy for us to interpolate them.
	for _, ref := range References {
		sort.Slice(ref.ReferencePoints, func(i, j int) bool {
			return ref.ReferencePoints[i].Value < ref.ReferencePoints[j].Value
		})
	}
}

// Find the reference point that matches the input value
func interpolateReferencePoints(refs []Service, value float64) Service {
	// Find a reference point below the value (a) and above the value (b).
	var (
		a, b  Service
		found bool
	)
	for i := range refs {
		if refs[i].Value >= value {
			if i > 0 {
				a = refs[i-1]
			} else {
				a = refs[i]
			}
			b = refs[i]
			found = true
			break
		}
	}
	if !found {
		// There is not a large enough reference point.
		ref := refs[len(refs)-1] // largest reference point
		ref.ContactSupport = true
		return ref
	}
	valueRange := b.Value - a.Value
	replicasRange := float64(b.Replicas - a.Replicas)
	scalingFactor := (value - a.Value) / orOne(valueRange)
	cpuRange := ResourceRange{Request: b.Resources.Requests.CPU, Limit: b.Resources.Limits.CPU}.Sub(ResourceRange{Request: a.Resources.Requests.CPU, Limit: a.Resources.Limits.CPU})
	memoryGBRange := ResourceRange{Request: b.Resources.Requests.MEM, Limit: b.Resources.Limits.MEM}.Sub(ResourceRange{Request: a.Resources.Requests.MEM, Limit: a.Resources.Limits.MEM})
	ephRange := ResourceRange{Request: b.Resources.Requests.EPH, Limit: b.Resources.Limits.EPH}.Sub(ResourceRange{Request: a.Resources.Requests.EPH, Limit: a.Resources.Limits.EPH})
	cpuValues := ResourceRange{Request: a.Resources.Requests.CPU, Limit: a.Resources.Limits.CPU}.Add(cpuRange.MulScalar(scalingFactor))
	memValues := ResourceRange{Request: a.Resources.Requests.MEM, Limit: a.Resources.Limits.MEM}.Add(memoryGBRange.MulScalar(scalingFactor))
	ephValues := ResourceRange{Request: a.Resources.Requests.EPH, Limit: a.Resources.Limits.EPH}.Add(ephRange.MulScalar(scalingFactor))
	return Service{
		NameInDocker: a.NameInDocker,
		Label:        a.Label,
		PodName:      a.PodName,
		Value:        a.Value * scalingFactor,
		Replicas:     a.Replicas + (int(math.Round(replicasRange * scalingFactor))),
		Resources: Resources{
			Requests: Resource{
				CPU: cpuValues.Request,
				MEM: memValues.Request,
				EPH: ephValues.Request,
			},
			Limits: Resource{
				CPU: cpuValues.Limit,
				MEM: memValues.Limit,
				EPH: ephValues.Limit,
			},
		},
		Storage: a.Storage,
	}
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

type Estimate struct {
	// inputs
	DeploymentType            string // calculated if set to "docker-compose"
	RecommendedDeploymentType string
	CodeInsight               string // If Code Insight is enabled
	EngagementRate            int    // The percentage of users who use Sourcegraph regularly.
	Repositories              int    // Number of repositories
	LargeMonorepos            int    // Number of monorepos - repos that are larger than 2GB (~50 times larger than the average size repo)
	LargestRepoSize           int    // Size of the largest repository in GB
	LargestIndexSize          int    // Size of the largest LSIF index file in GB
	TotalRepoSize             int    // Size of all repositories
	Users                     int    // Number of users

	// calculated results
	AverageRepositories int                        // Number of total repositories including monorepos: number repos + monorepos x 50
	ContactSupport      bool                       // Contact support required
	EngagedUsers        int                        // Number of users x engagement rate
	Services            map[string]Service         // List of services output
	DockerServices      map[string]DockerResources // List of services output for docker compose
	UserRepoSumRatio    int                        // The ratio used to determine deployment size:  (user count + average repos count) / 1000

	// These fields are the sum of the _requests_ of all services in the deployment, plus 50% of
	// the difference in limits. The thinking is that requests are often far too low as they do not
	// describe peak load of the service, and limits are often far too high
	// and the sweet spot is in the middle.
	TotalCPU, TotalMemoryGB, TotalStorageSize int

	TotalSharedCPU, TotalSharedMemoryGB int
}

func (e *Estimate) Calculate() *Estimate {
	e.EngagedUsers = e.Users
	e.UserRepoSumRatio = (e.Users + e.Repositories + e.LargeMonorepos*MonorepoFactor) / 1000
	e.AverageRepositories = e.Repositories + e.LargeMonorepos*MonorepoFactor
	e.Services = make(map[string]Service)
	e.DockerServices = make(map[string]DockerResources)
	for _, ref := range References {
		var value float64
		switch ref.ScalingFactor {
		case ByEngagedUsers:
			value = float64(e.Users)
		case ByAverageRepositories:
			value = float64(e.AverageRepositories)
		case ByLargeMonorepos:
			value = float64(e.LargeMonorepos)
		case ByLargestRepoSize:
			value = float64(e.LargestRepoSize)
		case ByLargestIndexSize:
			value = float64(e.LargestIndexSize)
		case ByUserRepoSumRatio:
			value = float64(e.UserRepoSumRatio)
		case ByTotalRepoSize:
			value = float64(e.TotalRepoSize)
		default:
			panic("never here")
		}
		v := interpolateReferencePoints(ref.ReferencePoints, value)
		if v.ContactSupport {
			e.ContactSupport = true
		}
		v.Label = ref.ServiceLabel
		v.PodName = ref.PodName
		v.NameInDocker = ref.DockerServiceName
		switch ref.ServiceName {
		case "codeinsights-db":
			if e.CodeInsight != "Enable" {
				v.Storage = float64(0)
			}
		case "codeintel-db":
			if e.LargestIndexSize == 0 {
				v.Storage = float64(0)
			}
		case "searcher":
			// MAX(Size of Largest + Size of All * 0.15, Size of All * 0.3)
			v.Resources.Requests.EPH = math.Max(float64(e.LargestRepoSize)+float64(e.TotalRepoSize)*0.15, float64(e.TotalRepoSize)*0.3)
			v.Resources.Limits.EPH = math.Max(float64(e.LargestRepoSize)+float64(e.TotalRepoSize)*0.3, float64(e.TotalRepoSize)*0.4)
		case "gitserver":
			// 30% More than the total repo size
			v.Storage = float64(e.TotalRepoSize * 130 / 100)
		case "minio":
			v.Storage = float64(e.LargestIndexSize)
		case "indexedSearch":
			v.Storage = float64(e.TotalRepoSize * 120 / 100 / 2)
		}
		r := e.Services[ref.ServiceName]
		(&r).join(&v)
		e.Services[ref.ServiceName] = r
		// create struct for docker-compose yaml file
		e.DockerServices[ref.DockerServiceName] = e.DockerServices[ref.ServiceName].join(&r)
	}
	if e.DeploymentType == "type" {
		e.DeploymentType = "kubernetes"
	}
	// Ensure we have the same replica counts for services that live in the
	// same pod.
	for _, pod := range pods {
		maxReplicas := 0
		for _, name := range pod {
			if replicas := e.Services[name].Replicas; replicas > maxReplicas {
				maxReplicas = replicas
			}
		}
		for _, name := range pod {
			v := e.Services[name]
			v.Replicas = maxReplicas
			e.Services[name] = v
		}
	}
	var (
		sumCPURequests, sumCPULimits, sumMemoryGBRequests, sumMemoryGBLimits, sumStorageSize float64
		largestCPULimit, largestMemoryGBLimit                                                float64
		visited                                                                              = map[string]struct{}{}
	)
	countRef := func(service string, ref *Service) {
		if _, ok := visited[service]; ok {
			return
		}
		visited[service] = struct{}{}
		sumStorageSize += ref.Storage
		sumCPURequests += ref.Resources.Requests.CPU
		sumCPULimits += ref.Resources.Limits.CPU
		sumMemoryGBRequests += ref.Resources.Requests.MEM
		sumMemoryGBLimits += ref.Resources.Limits.MEM
		if v := ref.Resources.Limits.CPU; v > largestCPULimit {
			largestCPULimit = v
		}
		if v := ref.Resources.Limits.MEM; v > largestMemoryGBLimit {
			largestMemoryGBLimit = v
		}
	}
	for service := range e.Services {
		r := e.Services[service]
		countRef(service, &r)
	}
	for service := range defaults {
		r := defaults[service][e.DeploymentType]
		countRef(service, &r)
	}
	e.RecommendedDeploymentType = "Docker Compose"
	if e.EngagedUsers <= 500 {
		e.TotalCPU = 8
	} else if e.EngagedUsers <= 1000 {
		e.TotalCPU = 16
	} else if e.EngagedUsers <= 5000 {
		e.TotalCPU = 32
	} else if e.EngagedUsers <= 10000 {
		e.TotalCPU = 48
	} else if e.EngagedUsers <= 20000 {
		e.TotalCPU = 96
	} else if e.EngagedUsers <= 40000 {
		e.TotalCPU = 192
		e.RecommendedDeploymentType = "Kubernetes with auto-scaling enabled"
	} else {
		e.TotalCPU = 260
		e.RecommendedDeploymentType = "Kubernetes with auto-scaling enabled"
	}
	if e.AverageRepositories <= 1000 {
		e.TotalMemoryGB = 32
	} else if e.AverageRepositories <= 10000 {
		e.TotalMemoryGB = 64
	} else if e.AverageRepositories <= 50000 {
		e.TotalMemoryGB = 128
	} else if e.AverageRepositories <= 100000 {
		e.TotalMemoryGB = 192
	} else if e.AverageRepositories <= 250000 {
		e.TotalMemoryGB = 384
	} else if e.AverageRepositories <= 500000 {
		e.TotalMemoryGB = 768
		e.RecommendedDeploymentType = "Kubernetes with auto-scaling enabled"
	} else {
		e.TotalMemoryGB = 1000
		e.RecommendedDeploymentType = "Kubernetes with auto-scaling enabled"
	}
	e.TotalStorageSize = int(math.Ceil(sumStorageSize))
	e.TotalSharedCPU = int(math.Ceil(largestCPULimit))
	e.TotalSharedMemoryGB = int(math.Ceil(largestMemoryGBLimit))
	return e
}
