package scaling

import (
	"bytes"
	"fmt"
	"math"
	"sort"
)

type Factor int

const (
	ByEngagedUsers        Factor = iota
	ByAverageRepositories Factor = iota
	ByLargeMonorepos      Factor = iota
	ByLargestRepoSize     Factor = iota
	ByLargestIndexSize    Factor = iota
	ByUserRepoSumRatio    Factor = iota
)

type Resource struct {
	Request, Limit float64
}

func (r Resource) Add(o Resource) Resource {
	return Resource{r.Request + o.Request, r.Limit + o.Limit}
}

func (r Resource) Sub(o Resource) Resource {
	return Resource{r.Request - o.Request, r.Limit - o.Limit}
}

func (r Resource) MulScalar(f float64) Resource {
	return Resource{r.Request * f, r.Limit * f}
}

func (r Resource) round() Resource {
	return Resource{resourceRound(r.Request), resourceRound(r.Limit)}
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

type ReferencePoint struct {
	// Value corresponding to the scaling factor type (users, repositories, large monorepos, etc.)
	Value float64

	// Optional values indicating that at the specified Value, these service properties are required.
	Replicas      int
	CPU, MemoryGB Resource

	// ContactSupport, when true, indicates that for the given value support should be contacted.
	ContactSupport bool
}

func (r ReferencePoint) join(o ReferencePoint) ReferencePoint {
	r.Value = 0
	if r.Replicas == 0 {
		r.Replicas = o.Replicas
	}
	if r.CPU == (Resource{}) {
		r.CPU = o.CPU
	}
	if r.MemoryGB == (Resource{}) {
		r.MemoryGB = o.MemoryGB
	}
	r.ContactSupport = r.ContactSupport || o.ContactSupport
	return r
}

func (r ReferencePoint) round() ReferencePoint {
	r.CPU = r.CPU.round()
	r.MemoryGB = r.MemoryGB.round()
	return r
}

type ServiceScale struct {
	ServiceName     string
	ScalingFactor   Factor
	ReferencePoints []ReferencePoint
}

type Range struct {
	Min, Max float64
}

const (
	// Heuristic which pretends 1 large monorepo == N average repositories.
	MonorepoFactor = 50
)

var (
	UsersRange               = Range{5, 10000}
	RepositoriesRange        = Range{5, 50000}
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

func interpolateReferencePoints(refs []ReferencePoint, value float64) ReferencePoint {
	// Find a reference point below the value (a) and above the value (b).
	var (
		a, b  ReferencePoint
		found bool
	)
	for i, ref := range refs {
		if ref.Value >= value {
			if i > 0 {
				a = refs[i-1]
			} else {
				a = refs[i]
			}
			b = ref
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
	cpuRange := b.CPU.Sub(a.CPU)
	memoryGBRange := b.MemoryGB.Sub(a.MemoryGB)

	return ReferencePoint{
		Value:    a.Value * scalingFactor,
		Replicas: a.Replicas + (int(math.Round(replicasRange * scalingFactor))),
		CPU:      a.CPU.Add(cpuRange.MulScalar(scalingFactor)),
		MemoryGB: a.MemoryGB.Add(memoryGBRange.MulScalar(scalingFactor)),
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
	Repositories     int
	LargeMonorepos   int
	TotalRepoSize    int
	LargestRepoSize  int
	LargestIndexSize int
	Users            int
	EngagementRate   int
	DeploymentType   string // calculated if set to "docker-compose"
	CodeIntel        string
	CodeInsight      string

	// calculated results
	UserRepoSumRatio    int
	EngagedUsers        int
	AverageRepositories int
	Services            map[string]ReferencePoint
	ContactSupport      bool

	// These fields are the sum of the _requests_ of all services in the deployment, plus 50% of
	// the difference in limits. The thinking is that requests are often far too low as they do not
	// describe peak load of the service, and limits are often far too high
	// and the sweet spot is in the middle.
	TotalCPU, TotalMemoryGB int

	TotalSharedCPU, TotalSharedMemoryGB int
}

func (e *Estimate) Calculate() *Estimate {
	e.EngagedUsers = e.Users * e.EngagementRate / 100
	e.UserRepoSumRatio = (e.Users + e.Repositories + e.LargeMonorepos*50) / 1000
	e.AverageRepositories = e.Repositories + (e.LargeMonorepos * MonorepoFactor)
	e.Services = make(map[string]ReferencePoint)
	for _, ref := range References {
		var value float64
		switch ref.ScalingFactor {
		case ByEngagedUsers:
			value = float64(e.EngagedUsers)
			if e.CodeInsight == "Yes" {
				value = float64(e.EngagedUsers + 2000)
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
		default:
			panic("never here")
		}
		v := interpolateReferencePoints(ref.ReferencePoints, value)
		if v.ContactSupport {
			e.ContactSupport = true
		}
		e.Services[ref.ServiceName] = e.Services[ref.ServiceName].join(v)
	}
	if e.DeploymentType == "type" {
		e.DeploymentType = "docker-compose"
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
		sumCPURequests, sumCPULimits, sumMemoryGBRequests, sumMemoryGBLimits float64
		largestCPULimit, largestMemoryGBLimit                                float64
		visited                                                              = map[string]struct{}{}
	)
	countRef := func(service string, ref ReferencePoint) {
		if _, ok := visited[service]; ok {
			return
		}
		visited[service] = struct{}{}
		sumCPURequests += ref.CPU.Request
		sumCPULimits += ref.CPU.Limit
		sumMemoryGBRequests += ref.MemoryGB.Request
		sumMemoryGBLimits += ref.MemoryGB.Limit
		if v := ref.CPU.Limit; v > largestCPULimit {
			largestCPULimit = v
		}
		if v := ref.MemoryGB.Limit; v > largestMemoryGBLimit {
			largestMemoryGBLimit = v
		}
	}
	for service, ref := range e.Services {
		countRef(service, ref)
	}
	for service, ref := range defaults[e.DeploymentType] {
		countRef(service, ref)
	}
	totalCPU := sumCPURequests + ((sumCPULimits - sumCPURequests) * 0.5)
	totalMemoryGB := sumMemoryGBRequests + ((sumMemoryGBLimits - sumMemoryGBRequests) * 0.5)
	e.TotalCPU = int(math.Ceil(totalCPU))
	e.TotalMemoryGB = int(math.Ceil(totalMemoryGB))
	e.TotalSharedCPU = int(math.Ceil(largestCPULimit))
	e.TotalSharedMemoryGB = int(math.Ceil(largestMemoryGBLimit))
	return e
}

func (e *Estimate) Markdown() []byte {
	var buf bytes.Buffer

	// Overview
	fmt.Fprintf(&buf, "### Estimate overview\n")
	fmt.Fprintf(&buf, "\n")
	var engagedUsers string
	if e.EngagedUsers > int(UsersRange.Max) {
		engagedUsers = fmt.Sprint(UsersRange.Max, "+")
	} else {
		engagedUsers = fmt.Sprint(e.EngagedUsers)
	}
	var averageRepositories string
	if e.AverageRepositories > int(AverageRepositoriesRange.Max) {
		averageRepositories = fmt.Sprint(AverageRepositoriesRange.Max, "+")
	} else {
		averageRepositories = fmt.Sprint(e.AverageRepositories)
	}
	fmt.Fprintf(&buf, "* Estimated resources for %v engaged users and %v average-size repositories.\n", engagedUsers, averageRepositories)
	if e.LargeMonorepos != 0 {
		fmt.Fprintf(&buf, "* Assuming 1 large monorepo is roughly equal to %v average-size repositories.\n", MonorepoFactor)
	}
	if e.DeploymentType == "docker-compose" {
		fmt.Fprintf(&buf, "* **Deployment type:** %v\n", e.DeploymentType)
	} else {
		fmt.Fprintf(&buf, "* <details><summary>**Deployment type:** %v</summary><br><blockquote>\n", e.DeploymentType)
		fmt.Fprintf(&buf, "  <p>We recommend Kubernetes for any deployments requiring > 1 service replica, but docker-compose does support service replicas and can scale up with multiple replicas as long as you can provision a suffiently large single machine.</p>\n")
		fmt.Fprintf(&buf, "  </blockquote></details>\n")
	}
	fmt.Fprintf(&buf, "* **Estimated total CPUs:** %v\n", e.TotalCPU)
	fmt.Fprintf(&buf, "* **Estimated total memory:** %vg\n", e.TotalMemoryGB)
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
	var totalRepoSize = e.TotalRepoSize * 2
	// Disk Spaces
	fmt.Fprintf(&buf, "### Disk Spaces\n")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "* Your gitserver disk space must be at least %v GB.\n", fmt.Sprint(totalRepoSize))
	fmt.Fprintf(&buf, "* Your pgsql disk space must be at least %v GB.\n", fmt.Sprint(totalRepoSize))
	fmt.Fprintf(&buf, "* Your minio disk space must be larger than %v GB.\n", fmt.Sprint(e.LargestIndexSize))

	fmt.Fprintf(&buf, "\n")

	// Service replicas & resources
	fmt.Fprintf(&buf, "### Service replicas & resources\n")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "| Service | Replicas | CPU requests | CPU limits | Memory requests | Memory limits | Note |\n")
	fmt.Fprintf(&buf, "|---------|----------|--------------|------------|-----------------|---------------|------|\n")

	var names []string
	for service := range e.Services {
		names = append(names, service)
	}
	sort.Strings(names)
	for _, service := range names {
		ref := e.Services[service]
		def := defaults[e.DeploymentType][service]
		ref = ref.round()
		plus := ""
		note := "-"

		if ref.ContactSupport {
			plus = "+"
			note = "[contact support](mailto:support@sourcegraph.com)"
		}

		var replicas string
		if ref.Replicas == def.Replicas {
			replicas = fmt.Sprint(ref.Replicas, plus)
		} else {
			replicas = fmt.Sprint("**_", ref.Replicas, plus, "ꜝ_**")
		}

		var cpuRequest string
		if ref.CPU.Request == def.CPU.Request {
			cpuRequest = fmt.Sprint(ref.CPU.Request, plus)
		} else {
			cpuRequest = fmt.Sprint("**_", ref.CPU.Request, plus, "ꜝ_**")
		}

		var cpuLimit string
		if ref.CPU.Limit == def.CPU.Limit {
			cpuLimit = fmt.Sprint(ref.CPU.Limit, plus)
		} else {
			cpuLimit = fmt.Sprint("**_", ref.CPU.Limit, plus, "ꜝ_**")
		}

		var memoryGBRequest string
		if ref.MemoryGB.Request == def.MemoryGB.Request {
			memoryGBRequest = fmt.Sprint(ref.MemoryGB.Request, "g", plus)
		} else {
			memoryGBRequest = fmt.Sprint("**_", ref.MemoryGB.Request, "g", plus, "ꜝ_**")
		}

		var memoryGBLimit string
		if ref.MemoryGB.Limit == def.MemoryGB.Limit {
			memoryGBLimit = fmt.Sprint(ref.MemoryGB.Limit, "g", plus)
		} else {
			memoryGBLimit = fmt.Sprint("**_", ref.MemoryGB.Limit, "g", plus, "ꜝ_**")
		}

		if e.DeploymentType == "docker-compose" {
			cpuRequest = "-"
			memoryGBRequest = "-"
		}

		fmt.Fprintf(
			&buf,
			"| %v | %v | %v | %v | %v | %v | %v |\n",
			service,
			replicas,
			cpuRequest,
			cpuLimit,
			memoryGBRequest,
			memoryGBLimit,
			note,
		)
	}
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "_Bold/italic and ꜝ indicate the value is modified from the default. Services not listed here use the default values._")
	return buf.Bytes()
}
