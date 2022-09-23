package scaling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
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
		r.Resources.Requests.CPUS = addUnit(r.Resources.Requests.CPU, "")
		r.Resources.Limits.CPUS = addUnit(r.Resources.Limits.CPU, "")
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
		r.PVC = addUnit(resourceRound(o.Storage/float64(r.Replicas)), "Gi")
	}
	r.ContactSupport = r.ContactSupport || o.ContactSupport
}

func (d DockerResources) join(o *Service) DockerResources {
	d.CPU = strings.ToLower(addUnit(o.Resources.Requests.CPU*float64(o.Replicas), ""))
	d.MEM = strings.ToLower(addUnit(o.Resources.Limits.MEM*float64(o.Replicas), "g"))
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
	MonorepoFactor = 50
)

var (
	UsersRange               = Range{5, 25000}
	RepositoriesRange        = Range{5, 5000000}
	TotalRepoSizeRange       = Range{1, 5000}
	LargeMonoreposRange      = Range{0, 10}
	LargestRepoSizeRange     = Range{0, 5000}
	LargestIndexSizeRange    = Range{0, 100}
	AverageRepositoriesRange = Range{
		RepositoriesRange.Min + (LargeMonoreposRange.Min * MonorepoFactor),
		RepositoriesRange.Max + (LargeMonoreposRange.Max * MonorepoFactor),
	}
	UserRepoSumRatioRange = Range{1, 200}
	EngagementRateRange   = Range{5, 100}
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
	DeploymentType   string // calculated if set to "docker-compose"
	CodeInsight      string // If Code Insight is enabled, add 1000 to user count
	EngagementRate   int    // The percentage of users who use Sourcegraph regularly.
	Repositories     int    // Number of repositories
	LargeMonorepos   int    // Number of monorepos - repos that are larger than 2GB (~50 times larger than the average size repo)
	LargestRepoSize  int    // Size of the largest repository in GB
	LargestIndexSize int    // Size of the largest LSIF index file in GB
	TotalRepoSize    int    // Size of all repositories
	Users            int    // Number of users

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
	e.EngagedUsers = e.Users * e.EngagementRate / 100
	e.UserRepoSumRatio = (e.Users + e.Repositories + e.LargeMonorepos*MonorepoFactor) / 1000
	e.AverageRepositories = e.Repositories + e.LargeMonorepos*MonorepoFactor
	e.Services = make(map[string]Service)
	e.DockerServices = make(map[string]DockerResources)
	for _, ref := range References {
		var value float64
		switch ref.ScalingFactor {
		case ByEngagedUsers:
			value = float64(e.EngagedUsers)
			if e.CodeInsight == "Enable" {
				value = float64(e.EngagedUsers + 1000)
			}
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
		// calculate storage size
		var dockerFactor = 1
		var k8sFactor = 0
		if e.DeploymentType == "docker-compose" {
			dockerFactor = 2
			k8sFactor = 1
		}
		switch ref.ServiceName {
		case "gitserver":
			v.Storage = float64(e.TotalRepoSize * 120 / 100)
		case "minio":
			v.Storage = float64(e.LargestIndexSize)
		case "indexedSearch":
			v.Storage = float64(e.TotalRepoSize * 120 / 100 / 2 / dockerFactor) // times 2 for docker compose deployment
		case "indexedSearchIndexer":
			v.Storage = float64(e.TotalRepoSize * 120 / 100 / 2 / dockerFactor * k8sFactor) // zero for k8s deployment
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
	totalCPU := sumCPURequests + ((sumCPULimits - sumCPURequests) * 0.5)
	totalMemoryGB := sumMemoryGBRequests + ((sumMemoryGBLimits - sumMemoryGBRequests) * 0.5)
	e.TotalStorageSize = int(math.Ceil(sumStorageSize))
	e.TotalCPU = int(math.Ceil(totalCPU))
	e.TotalMemoryGB = int(math.Ceil(totalMemoryGB))
	e.TotalSharedCPU = int(math.Ceil(largestCPULimit))
	e.TotalSharedMemoryGB = int(math.Ceil(largestMemoryGBLimit))
	return e
}

func (e *Estimate) MarkdownExport() []byte {
	var buf bytes.Buffer
	// Summary of the output
	fmt.Fprintf(&buf, "### Estimate summary\n")
	fmt.Fprintf(&buf, "\n")
	if !e.ContactSupport {
		fmt.Fprintf(&buf, "* **Estimated total CPUs:** %v\n", e.TotalCPU)
		fmt.Fprintf(&buf, "* **Estimated total memory:** %vg\n", e.TotalMemoryGB)
		fmt.Fprintf(&buf, "* **Estimated total storage:** %vg\n", e.TotalStorageSize)
	} else {
		fmt.Fprintf(&buf, "* **Estimated total CPUs:** not available\n")
		fmt.Fprintf(&buf, "* **Estimated total memory:** not available\n")
		fmt.Fprintf(&buf, "* **Estimated total storage:** not available\n")
	}
	fmt.Fprintf(&buf, "\n<small>**Note:** The estimated total includes default values for other services.</small>\n")
	if e.EngagedUsers < 650/2 && e.AverageRepositories < 1500/2 {
		if e.DeploymentType == "docker-compose" {
			fmt.Fprintf(&buf, "* <details><summary>**IMPORTANT:** Cost-saving option to reduce resource consumption is available</summary><br><blockquote>\n")
			fmt.Fprintf(&buf, "  <p>You may choose to use _shared resources_ to reduce the costs of your deployment:</p>\n")
			fmt.Fprintf(&buf, "  <ul>\n")
			fmt.Fprintf(&buf, "  <li>**Estimated total _shared_ CPUs (shared):** %v</li>\n", e.TotalSharedCPU)
			fmt.Fprintf(&buf, "  <li>**Estimated total _shared_ memory (shared):** %vg</li>\n", e.TotalSharedMemoryGB)
			fmt.Fprintf(&buf, "  </ul><br>\n")
			fmt.Fprintf(&buf, "  <p>**What this means:** Your instance would not have enough resources for all services to run _at peak load_, and _sometimes_ this could lead to a lack of resources. This may appear as searches being slow for some users if many other requests or indexing jobs are ongoing.</p>\n")
			fmt.Fprintf(&buf, "  <p>On small instances such as what you've chosen, this can often be OK -- just keep an eye out for any performance issues and increase resources as needed.</p>\n")
			fmt.Fprintf(&buf, "  <p>To use shared resources, simply apply the limits shown below normally -- but only provision a machine with the resources shown above.</p>\n")
			fmt.Fprintf(&buf, "  </blockquote></details>\n")
		} else if e.DeploymentType == "kubernetes" {
			fmt.Fprintf(&buf, "* <details><summary>**IMPORTANT:** Cost-saving option to reduce resource consumption is available</summary><br><blockquote>\n")
			fmt.Fprintf(&buf, "  <p>You may choose to use _shared resources_ to reduce the costs of your deployment:</p>\n")
			fmt.Fprintf(&buf, "  <ul>\n")
			fmt.Fprintf(&buf, "  <li>**Estimated total _shared_ CPUs (shared):** %v</li>\n", e.TotalSharedCPU)
			fmt.Fprintf(&buf, "  <li>**Estimated total _shared_ memory (shared):** %vg</li>\n", e.TotalSharedMemoryGB)
			fmt.Fprintf(&buf, "  </ul><br>\n")
			fmt.Fprintf(&buf, "  <p>**What this means:** Your instance would not have enough resources for all services to run _at peak load_, and _sometimes_ this could lead to a lack of resources. This may appear as searches being slow for some users if many other requests or indexing jobs are ongoing.</p>\n")
			fmt.Fprintf(&buf, "  <p>On small instances such as what you've chosen, this can often be OK -- just keep an eye out for any performance issues and increase resources as needed.</p>\n")
			fmt.Fprintf(&buf, "  <p>To use shared resources, simply apply the \"limits\" shown below normally and remove or reduce the \"requests\" for each service.</p>\n")
			fmt.Fprintf(&buf, "  </blockquote></details>\n")
		}
	}
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "\n")
	if e.ContactSupport {
		fmt.Fprintf(&buf, "> **IMPORTANT:** Please [contact support](mailto:support@sourcegraph.com) for the service(s) that marked as not available.\n")
		fmt.Fprintf(&buf, "\n")
	}
	fmt.Fprintf(&buf, "| Service | Replica | CPU requests | CPU limits | MEM requests | MEM limits | EPH requests | EPH limits | Storage |\n")
	fmt.Fprintf(&buf, "|-------|:-------:|:-------:|:-------:|:-------:|:-------:|:-------:|:-------:|:-------:|\n")

	var names []string
	for service := range e.Services {
		names = append(names, service)
	}
	sort.Strings(names)

	for _, service := range names {
		ref := e.Services[service]
		def := defaults[service][e.DeploymentType]
		plus := ""
		serviceName := fmt.Sprint(ref.Label, "</br><small>(pod: ", ref.PodName, ")</small>")
		replicas := "n/a"
		cpuRequest := "n/a"
		cpuLimit := "n/a"
		memoryGBRequest := "n/a"
		memoryGBLimit := "n/a"
		ephRequest := "-"
		ephLimit := "-"
		pvc := "-"

		if !ref.ContactSupport {
			if e.DeploymentType == "docker-compose" {
				serviceName = ref.NameInDocker
				replicas = fmt.Sprint("1", plus)
				cpuRequest = "-"
				cpuLimit = fmt.Sprint(ref.Resources.Limits.CPU*float64(ref.Replicas), plus)
				memoryGBRequest = "-"
				memoryGBLimit = fmt.Sprint(ref.Resources.Limits.MEM*float64(ref.Replicas), "g", plus)
				ephRequest = "-"
				ephLimit = "-"
				if ref.Storage > 0 {
					pvc = fmt.Sprint(ref.Storage, "G", plus, "ꜝ")
				}
				if ref.Resources.Limits.EPH > 0 {
					pvc = fmt.Sprint(ref.Resources.Limits.EPH, "G", plus, "ꜝ")
				}
			}
			if e.DeploymentType == "kubernetes" {
				replicas = fmt.Sprint(ref.Replicas, plus)
				cpuRequest = fmt.Sprint(ref.Resources.Requests.CPU, plus)
				cpuLimit = fmt.Sprint(ref.Resources.Limits.CPU, plus)
				memoryGBRequest = fmt.Sprint(ref.Resources.Requests.MEMS, plus)
				memoryGBLimit = fmt.Sprint(ref.Resources.Limits.MEMS, plus)
				if ref.Replicas != def.Replicas {
					replicas += "ꜝ"
				}
				if ref.Resources.Requests.CPU != def.Resources.Requests.CPU {
					cpuRequest += "ꜝ"
				}
				if ref.Resources.Requests.MEM != def.Resources.Requests.MEM {
					memoryGBRequest += "ꜝ"
				}
				if ref.Storage > 0 {
					pvc = fmt.Sprint(ref.PVC, plus, "ꜝ")
				}
				if ref.Resources.Limits.EPH > 0 {
					ephRequest = fmt.Sprint(ref.Resources.Requests.EPHS, plus, "ꜝ")
					ephLimit = fmt.Sprint(ref.Resources.Limits.EPHS, plus, "ꜝ")
				}
			}
			if ref.Resources.Limits.CPU != def.Resources.Limits.CPU {
				cpuLimit += "ꜝ"
			}
			if ref.Resources.Limits.MEM != def.Resources.Limits.MEM {
				memoryGBLimit += "ꜝ"
			}
		}
		fmt.Fprintf(
			&buf,
			"| %v | %v | %v | %v | %v | %v | %v | %v | %v |\n",
			serviceName,
			replicas,
			cpuRequest,
			cpuLimit,
			memoryGBRequest,
			memoryGBLimit,
			ephRequest,
			ephLimit,
			pvc,
		)
	}
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "> ꜝ<small> This is a non-default value.</small>\n")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "\n")

	return buf.Bytes()
}

func (e *Estimate) HelmExport() string {
	var c = e.Services
	j, err := json.Marshal(c)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	s := strings.Replace(string(y), `"`, "'", -1)
	return s
}

func (e *Estimate) DockerExport() string {
	var d DockerServices
	d.Version = "2.4"
	d.Services = e.DockerServices
	j, err := json.Marshal(d)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	s := strings.Replace(string(y), `"`, "", -1)
	return s
}
