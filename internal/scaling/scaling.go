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
	ByTotalRepoSize       Factor = iota
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
	LargeMonoreposRange      = Range{0, 20}
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
	DeploymentType   string // calculated if set to "docker-compose"
	CodeIntel        string
	CodeInsight      string
	EngagementRate   int
	Repositories     int
	LargeMonorepos   int
	LargestRepoSize  int
	LargestIndexSize int
	TotalRepoSize    int
	Users            int

	// calculated results
	AverageRepositories int
	ContactSupport      bool
	EngagedUsers        int
	IndexServerDiskSize int
	Services            map[string]ReferencePoint
	UserRepoSumRatio    int

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
			if e.CodeInsight == "Enable" {
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
		case ByTotalRepoSize:
			value = float64(e.TotalRepoSize)
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
		e.DeploymentType = "kubernetes"
	}
	// Get index server disk size
	// Typically the gitserver disk size multipled by the number of gitserver shards
	// formula: size of gitserver (size of all repos x 120%) x gitserver replicas
	// ref: https://github.com/sourcegraph/deploy-sourcegraph/blob/v3.41.0/base/indexed-search/indexed-search.StatefulSet.yaml#L84
	switch {
	case e.UserRepoSumRatio < 5:
		e.IndexServerDiskSize = e.TotalRepoSize * 120 / 100
	case e.UserRepoSumRatio < 20:
		e.IndexServerDiskSize = e.TotalRepoSize * 120 / 100 * 2
	case e.UserRepoSumRatio < 30:
		e.IndexServerDiskSize = e.TotalRepoSize * 120 / 100 * 3
	case e.UserRepoSumRatio < int(UserRepoSumRatioRange.Max):
		e.IndexServerDiskSize = e.TotalRepoSize * 120 / 100 * 4
	default:
		e.IndexServerDiskSize = e.TotalRepoSize * 120 / 100 * 5
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

func (e *Estimate) Result() []byte {
	var buf bytes.Buffer
	// Summary of the output
	fmt.Fprintf(&buf, "### Estimate summary\n")
	fmt.Fprintf(&buf, "\n")
	if !e.ContactSupport {
		fmt.Fprintf(&buf, "* **Estimated total CPUs:** %v\n", e.TotalCPU)
		fmt.Fprintf(&buf, "* **Estimated total memory:** %vg\n", e.TotalMemoryGB)
	} else {
		fmt.Fprintf(&buf, "* **Estimated total CPUs:** not available\n")
		fmt.Fprintf(&buf, "* **Estimated total memory:** not available\n")
	}
	fmt.Fprintf(&buf, "* **Note:** Use the default values for services not listed below .\n")
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
	fmt.Fprintf(&buf, "| Service | Replicas | CPU requests | CPU limits | Memory requests | Memory limits |\n")
	fmt.Fprintf(&buf, "|---------|:----------:|:--------------:|:------------:|:-----------------:|:---------------:|\n")

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

		var replicas = "n/a"
		var cpuRequest = "n/a"
		var cpuLimit = "n/a"
		var memoryGBRequest = "n/a"
		var memoryGBLimit = "n/a"

		if !ref.ContactSupport {
			if ref.Replicas == def.Replicas {
				replicas = fmt.Sprint(ref.Replicas, plus)
			} else {
				replicas = fmt.Sprint(ref.Replicas, plus, "ꜝ")
			}

			if ref.CPU.Request == def.CPU.Request {
				cpuRequest = fmt.Sprint(ref.CPU.Request, plus)
			} else {
				cpuRequest = fmt.Sprint(ref.CPU.Request, "ꜝ")
			}

			if ref.CPU.Limit == def.CPU.Limit {
				cpuLimit = fmt.Sprint(ref.CPU.Limit, plus)
			} else {
				cpuLimit = fmt.Sprint(ref.CPU.Limit, plus, "ꜝ")
			}

			if ref.MemoryGB.Request == def.MemoryGB.Request {
				memoryGBRequest = fmt.Sprint(ref.MemoryGB.Request, "g", plus)
			} else {
				memoryGBRequest = fmt.Sprint("", ref.MemoryGB.Request, "g", plus, "ꜝ")
			}

			if ref.MemoryGB.Limit == def.MemoryGB.Limit {
				memoryGBLimit = fmt.Sprint(ref.MemoryGB.Limit, "g", plus)
			} else {
				memoryGBLimit = fmt.Sprint(ref.MemoryGB.Limit, "g", plus, "ꜝ")
			}

			if e.DeploymentType == "docker-compose" {
				cpuRequest = "-"
				memoryGBRequest = "-"
			}
		}

		fmt.Fprintf(
			&buf,
			"| %v | %v | %v | %v | %v | %v |\n",
			service,
			replicas,
			cpuRequest,
			cpuLimit,
			memoryGBRequest,
			memoryGBLimit,
		)
	}
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "> ꜝ<small>: This is a non-default value.</small>\n")
	fmt.Fprintf(&buf, "\n")

	// Storage Size
	fmt.Fprintf(&buf, "### Storage\n")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "| Service | Size | Note |\n")
	fmt.Fprintf(&buf, "|---------|:------------:|------|\n")
	fmt.Fprintf(&buf, "| codeinsights-db | 200GB | Starts at default value as the value depends entirely on usage and the specific Insights that are being created by users. |\n")
	fmt.Fprintf(&buf, "| codeintel-db | 200GB | Starts at default value as the value depends entirely on the size of indexes being uploaded. If Rockskip is enabled, 4 times the size of all repos indexed by Rockskip is required. |\n")
	fmt.Fprintf(&buf, "| gitserver | %v | At least 20 percent more than the total size of all repoes. |\n", fmt.Sprint(float64(e.TotalRepoSize*120/100), "GB"))
	fmt.Fprintf(&buf, "| minio | %v | The size of the largest LSIF file. |\n", fmt.Sprint(e.LargestIndexSize, "GB"))
	fmt.Fprintf(&buf, "| pgsql | %v | Two times the size of your current database is required for migration. |\n", fmt.Sprint(e.TotalRepoSize*2, "GB"))
	fmt.Fprintf(&buf, "| indexed-search | %v | The disk size for gitserver multipled by the number of gitserver replicas. |\n", fmt.Sprint(e.IndexServerDiskSize, "GB"))

	fmt.Fprintf(&buf, "\n")

	// Ephemeral Storage
	fmt.Fprintf(&buf, "### Ephemeral storage\n")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "| Service | Size | Note |\n")
	fmt.Fprintf(&buf, "|---------|:------------:|------|\n")
	fmt.Fprintf(&buf, "| searcher| %v | The size of all indexed repos. |\n", fmt.Sprint(float64(e.TotalRepoSize*30/100), "GB"))
	fmt.Fprintf(&buf, "| symbols | %v | At least 20 percent more than the size of your largest repo. Using an SSD is highly preferred if you are not indexing with Rockskip. |\n", fmt.Sprint(float64(e.LargestRepoSize*120/100), "GB"))

	fmt.Fprintf(&buf, "\n")

	// Scaling overview
	fmt.Fprintf(&buf, "## Service overview\n")
	fmt.Fprintf(&buf, "\n")
	fmt.Fprintf(&buf, "| Service | Description |\n")
	fmt.Fprintf(&buf, "|---------|------|\n")
	fmt.Fprintf(&buf, "| cadvisor | A cAdvisor instance that exports container monitoring metrics scraped by Prometheus and visualized in Grafana |\n")
	fmt.Fprintf(&buf, "| codeinsights-db | A PostgreSQL instance for storing code insights data |\n")
	fmt.Fprintf(&buf, "| codeintel-db | A PostgreSQL instance for storing large-volume precise code intelligence data |\n")
	fmt.Fprintf(&buf, "| frontend | Serves the web application, extensions, and graphQL services. Almost every service has a link back to the frontend, from which it gathers configuration updates. |\n")
	fmt.Fprintf(&buf, "| github-proxy | Proxies all requests to github.com to keep track of rate limits and prevent triggering abuse mechanisms	|\n")
	fmt.Fprintf(&buf, "| gitserver | Mirrors repositories from their code host. All other Sourcegraph services talk to gitserver when they need data from git  |\n")
	fmt.Fprintf(&buf, "| grafana | A Grafana instance that displays data from Prometheus and Jaeger. It is shipped with customized dashboards for Sourcegraph services |\n")
	fmt.Fprintf(&buf, "| jaeger | A Jaeger instance for end-to-end distributed tracing |\n")
	fmt.Fprintf(&buf, "| minio | A MinIO instance that serves as a local S3-compatible object storage to hold user uploads for code-intel before they can be processed |\n")
	fmt.Fprintf(&buf, "| pgsql | The main database. It is a PostgreSQL instance where things like repo lists, user data, site config files are stored (anything not related to code-intel and code-insights) |\n")
	fmt.Fprintf(&buf, "| precise-code-intel | Converts LSIF upload file into Postgres data. The entire index must be read into memory to be correlated |\n")
	fmt.Fprintf(&buf, "| prometheus | Collecting high-level, and low-cardinality, metrics across services. |\n")
	fmt.Fprintf(&buf, "| redis-cache | A Redis instance for storing cache data. |\n")
	fmt.Fprintf(&buf, "| redis-store  | A Redis instance for storing short-term information such as user sessions. |\n")
	fmt.Fprintf(&buf, "| repo-updater | Repo-updater tracks the state of repos, and is responsible for automatically scheduling updates using gitserver. Other apps which desire updates or fetches should be telling repo-updater, rather than using gitserver directly, so repo-updater can take their changes into account. |\n")
	fmt.Fprintf(&buf, "| searcher | Provides on-demand unindexed search for repositories. It fetches archives from gitserver and searches them with regexp	|\n")
	fmt.Fprintf(&buf, "| symbols | Indexes symbols in repositories using Ctags. By default, the symbols service saves SQLite DBs as files on disk, and copies an old one to a new file when a user visits a new commit. If Rockskip is enabled, the symbols are stored in the codeintel-db instead while the cache is stored on disk |\n")
	fmt.Fprintf(&buf, "| syntect-server | An HTTP server that exposes the Rust Syntect syntax highlighting library for use by other services |\n")
	fmt.Fprintf(&buf, "| worker |  Runs a collection of background jobs (for both Code-Intel and Code-Insight) periodically or in response to an external event. It is currently janitorial and commit based. |\n")
	fmt.Fprintf(&buf, "| zoekt-indexserver | Indexes all enabled repositories on Sourcegraph, as well as keeping the indexes up to date |\n")
	fmt.Fprintf(&buf, "| zoekt-webserver | Runs searches from in-memory indexes, but persists these indexes to disk to avoid re-indexing everything on startup |\n")
	fmt.Fprintf(&buf, "\n")

	return buf.Bytes()
}
