package scaling

// We are using the data gathered from different existing deployments as references for the estimates:
// https://docs.google.com/spreadsheets/d/1N7X_OXDwKk0QSR2Ghbj7ZhjVrQXcMNj-yC8mF1amBi4/edit?usp=sharing

var References = []ServiceScale{
	// Frontend scales based on the number of engaged users.
	// Add 2000 users to user count if code-insight is enabled
	{
		ServiceName:       "frontend",
		ServiceLabel:      "sourcegraph-frontend",
		DockerServiceName: "sourcegraph-frontend-0",
		PodName:           "frontend",
		ScalingFactor:     ByEngagedUsers, // UsersRange = {5, 10000}
		ReferencePoints: []Service{
			{Replicas: 5, Resources: Resources{Requests: Resource{CPU: 2, MEM: 8}, Limits: Resource{CPU: 4, MEM: 16}}, Value: UsersRange.Max}, // estimate
			{Replicas: 3, Resources: Resources{Requests: Resource{CPU: 4, MEM: 8}, Limits: Resource{CPU: 8, MEM: 16}}, Value: 5000},           // estimate
			{Replicas: 3, Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 8}}, Value: 2100},            // existing deployment: #4
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 8}}, Value: 2050},            // existing deployment: #45
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: UsersRange.Min},  // default for instance with <2000 users without code-insight
		},
	},

	// Gitserver scales based on the total size of all repoes and number of average repositories.
	{
		ServiceName:       "gitserver",
		ServiceLabel:      "gitserver",
		DockerServiceName: "gitserver-0",
		PodName:           "gitserver",
		ScalingFactor:     ByUserRepoSumRatio,
		ReferencePoints: []Service{
			{Replicas: 5, Resources: Resources{Requests: Resource{CPU: 16, MEM: 32}, Limits: Resource{CPU: 16, MEM: 32}}, Value: UserRepoSumRatioRange.Max}, // estimate
			{Replicas: 4, Resources: Resources{Requests: Resource{CPU: 16, MEM: 32}, Limits: Resource{CPU: 16, MEM: 32}}, Value: 150},                       // estimate
			{Replicas: 4, Resources: Resources{Requests: Resource{CPU: 8, MEM: 16}, Limits: Resource{CPU: 8, MEM: 16}}, Value: 50},                          // existing deployment: dogfood
			{Replicas: 3, Resources: Resources{Requests: Resource{CPU: 8, MEM: 32}, Limits: Resource{CPU: 8, MEM: 32}}, Value: 30},                          // estimate
			{Replicas: 3, Resources: Resources{Requests: Resource{CPU: 8, MEM: 16}, Limits: Resource{CPU: 8, MEM: 16}}, Value: 20},                          // estimate
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 8, MEM: 32}, Limits: Resource{CPU: 8, MEM: 32}}, Value: 10},                          // estimate
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 8, MEM: 16}, Limits: Resource{CPU: 8, MEM: 16}}, Value: 5},                           // estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 8, MEM: 16}, Limits: Resource{CPU: 8, MEM: 16}}, Value: 2},                           // estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 8}, Limits: Resource{CPU: 4, MEM: 8}}, Value: UserRepoSumRatioRange.Min},     // default for instance with <4000 repos
		},
	},

	{
		ServiceName:       "minio",
		ServiceLabel:      "minio",
		DockerServiceName: "minio",
		PodName:           "minio",
		ScalingFactor:     ByLargestIndexSize,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: .5}, Limits: Resource{CPU: 1, MEM: .5}}, Storage: LargestIndexSizeRange.Max, Value: LargestIndexSizeRange.Max}, // calculation
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: .5}, Limits: Resource{CPU: 1, MEM: .5}}, Storage: LargestIndexSizeRange.Min, Value: LargestIndexSizeRange.Min}, // bare minimum
		},
	},

	// Memory usage depends on the number of active users and service-connections
	{
		ServiceName:       "pgsql",
		ServiceLabel:      "pgsql",
		DockerServiceName: "pgsql",
		PodName:           "pgsql",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 7, MEM: 32}, Limits: Resource{CPU: 7, MEM: 32}}, Value: AverageRepositoriesRange.Max}, // existing deployment: dogfood
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 16}, Limits: Resource{CPU: 4, MEM: 16}}, Value: 25000},                        // existing deployment: #4
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 8}, Limits: Resource{CPU: 4, MEM: 8}}, Value: 4000},                           // existing deployment: #43
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Value: AverageRepositoriesRange.Min},   // bare minimum
		},
	},

	// The entire index must be read into memory to be correlated.
	// Scale vertically when the uploaded index is too large to be processed without OOMing the worker.
	// Scale horizontally to process a higher throughput of indexes.
	// calculation: ~2 times of the size of the largest index
	{
		ServiceName:       "preciseCodeIntel",
		ServiceLabel:      "precise-code-intel-worker",
		DockerServiceName: "precise-code-intel-worker",
		PodName:           "precise-code-intel",
		ScalingFactor:     ByLargestIndexSize,
		ReferencePoints: []Service{
			{Replicas: 4, Resources: Resources{Requests: Resource{CPU: 2, MEM: 25}, Limits: Resource{CPU: 4, MEM: 50}}, Value: LargestIndexSizeRange.Max}, // calculation
			{Replicas: 4, Resources: Resources{Requests: Resource{CPU: 2, MEM: 20}, Limits: Resource{CPU: 4, MEM: 41}}, Value: 81},                        // calculation
			{Replicas: 3, Resources: Resources{Requests: Resource{CPU: 2, MEM: 29}, Limits: Resource{CPU: 4, MEM: 58}}, Value: 80},                        // calculation
			{Replicas: 3, Resources: Resources{Requests: Resource{CPU: 2, MEM: 20}, Limits: Resource{CPU: 4, MEM: 40}}, Value: 61},                        // calculation
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 30}, Limits: Resource{CPU: 4, MEM: 60}}, Value: 60},                        // calculation
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 16}, Limits: Resource{CPU: 4, MEM: 32}}, Value: 32},                        // calculation
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 8}}, Value: 8},                           // calculation
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 8}, Limits: Resource{CPU: 4, MEM: 16}}, Value: 7},                          // calculation
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: 1},                          // default
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: LargestIndexSizeRange.Min},  // bare minimum
		},
	},

	// Searcher replicas scale based the number of concurrent unidexed queries & number concurrent of structural searches
	{
		ServiceName:       "searcher",
		ServiceLabel:      "searcher",
		DockerServiceName: "searcher-0",
		PodName:           "searcher",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 4, Value: AverageRepositoriesRange.Max}, // existing deployment: dogfood
			{Replicas: 1, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},
	// Searcher is IO and CPU bound. It fetches archives from gitserver and searches them with regexp.
	// Memory scales based on the size of repositories (i.e. when large monorepos are in the picture).
	{
		ServiceName:       "searcher",
		ServiceLabel:      "searcher",
		DockerServiceName: "searcher-0",
		PodName:           "searcher",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{CPU: 3, MEM: 4, EPH: 440}, Limits: Resource{CPU: 6, MEM: 8, EPH: 480}}, Value: AverageRepositoriesRange.Max}, // estimate. eph based on dogfood
			{Resources: Resources{Requests: Resource{CPU: 3, MEM: 4, EPH: 220}, Limits: Resource{CPU: 6, MEM: 8, EPH: 240}}, Value: 25000},                        // existing deployment: #4
			{Resources: Resources{Requests: Resource{CPU: .5, MEM: 2, EPH: 25}, Limits: Resource{CPU: 2, MEM: 4, EPH: 26}}, Value: 4000},                          // existing deployment: #43
			{Resources: Resources{Requests: Resource{CPU: .5, MEM: .5, EPH: 25}, Limits: Resource{CPU: 2, MEM: 2, EPH: 26}}, Value: AverageRepositoriesRange.Min}, // default
		},
	},

	// Symbols replicas scale based on the number of average repositories, and its resources scale
	// based on the size of repositories (i.e. when large monorepos are in the picture).
	{
		ServiceName:       "symbols",
		ServiceLabel:      "symbols",
		DockerServiceName: "symbols-0",
		PodName:           "symbols",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 4, Value: AverageRepositoriesRange.Max}, // estimate
			{Replicas: 3, Value: 25000},                        // estimate
			{Replicas: 2, Value: 4000},                         // estimate
			{Replicas: 1, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},
	{
		ServiceName:       "symbols",
		ServiceLabel:      "symbols",
		DockerServiceName: "symbols-0",
		PodName:           "symbols",
		ScalingFactor:     ByLargeMonorepos,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{CPU: 2, MEM: 8}, Limits: Resource{CPU: 4, MEM: 16}}, Value: LargeMonoreposRange.Max},  // estimate
			{Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 8}}, Value: 4},                         // estimate
			{Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: 2},                        // existing deployment: #43
			{Resources: Resources{Requests: Resource{CPU: .5, MEM: .5}, Limits: Resource{CPU: 2, MEM: 2}}, Value: LargeMonoreposRange.Min}, // default
		},
	},
	{
		ServiceName:       "symbols",
		ServiceLabel:      "symbols",
		DockerServiceName: "symbols-0",
		PodName:           "symbols",
		ScalingFactor:     ByLargestRepoSize,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{EPH: 5900}, Limits: Resource{EPH: 6000}}, Value: LargestRepoSizeRange.Max}, // calculation
			{Resources: Resources{Requests: Resource{EPH: 110}, Limits: Resource{EPH: 120}}, Value: 100},                        // calculation
			{Resources: Resources{Requests: Resource{EPH: 50}, Limits: Resource{EPH: 60}}, Value: 50},                           // calculation
			{Resources: Resources{Requests: Resource{EPH: 5}, Limits: Resource{EPH: 6}}, Value: 5},                              // calculation
			{Resources: Resources{Requests: Resource{EPH: 2}, Limits: Resource{EPH: 3}}, Value: 2},                              // calculation
			{Resources: Resources{Requests: Resource{EPH: 1}, Limits: Resource{EPH: 2}}, Value: LargestRepoSizeRange.Min},       // bare minimum
		},
	},

	// At initialization time, many highlighting themes and compiled grammars are loaded into memory.
	// There is additional memory consumption on receiving requests (< 25 MB), although,
	// that's generally much smaller than the constant overhead (1-2 GB).
	// In some situations, there are hangs with syntax highlighting.
	// These can cause runaway CPU usage (for 1 core per hang).
	// syntect-server should normally kill such processes and restart them if that happens.
	{
		ServiceName:       "syntectServer",
		ServiceLabel:      "syntect-server",
		DockerServiceName: "syntect-server",
		PodName:           "syntect-server",
		ScalingFactor:     ByEngagedUsers,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 6}, Limits: Resource{CPU: 8, MEM: 12}}, Value: UsersRange.Max}, // estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 5000},           // existing deployment: average between 27 and
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 4, MEM: 6}}, Value: UsersRange.Min}, // default
		},
	},

	// worker is used by different services, and mostly scale based on the number of average repositories to execute jobs
	{
		ServiceName:       "worker",
		ServiceLabel:      "worker",
		DockerServiceName: "worker",
		PodName:           "worker",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 8}, Limits: Resource{CPU: 4, MEM: 16}}, Value: AverageRepositoriesRange.Max}, // estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 8}}, Value: 25000},                         // estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: AverageRepositoriesRange.Min}, // default
		},
	},

	// zoekt-indexserver memory usage scales based on whether it must index large monorepos
	{
		ServiceName:       "indexedSearch",
		ServiceLabel:      "zoekt-indexserver",
		DockerServiceName: "zoekt-indexserver-0",
		PodName:           "indexed-search",
		ScalingFactor:     ByLargeMonorepos,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 16}}, Value: LargeMonoreposRange.Max}, // estimate
			{Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 16}}, Value: 2},                       // estimate
			{Resources: Resources{Requests: Resource{MEM: 4}, Limits: Resource{MEM: 8}}, Value: LargeMonoreposRange.Min},  // default
		},
	},
	// CPU usage and replicas scale based on the number of average repos it must index as it indexes one repo at a time
	// Set replica number to 0 as it will be synced with the replica number for webserver
	{
		ServiceName:       "indexedSearch",
		ServiceLabel:      "zoekt-indexserver",
		DockerServiceName: "zoekt-indexserver-0",
		PodName:           "indexed-search",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 4, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: AverageRepositoriesRange.Max}, // estimate: 4 replics to serve 50k repos so 8CPU limit per replica should be enough
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: 14000},                        // existing deployment: #26 - 16 CPU / 2 replicas = 8
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: 10000},                        // existing deployment: #37
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: 1500},                         // existing deployment: #44
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: AverageRepositoriesRange.Min}, // default
		},
	},

	// zoekt-webserver memory usage and replicas scale based on how many average repositories it is
	// serving (roughly 2/3 the size of the actual repos is the memory usage).
	{
		ServiceName:       "indexedSearchIndexer",
		DockerServiceName: "zoekt-webserver-0",
		ServiceLabel:      "zoekt-webserver",
		PodName:           "indexed-search",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 4, Resources: Resources{Requests: Resource{MEM: 25}, Limits: Resource{MEM: 50}}, Value: AverageRepositoriesRange.Max}, // existing deployment: dogfood
			{Replicas: 2, Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 16}}, Value: 14000},                         // existing deployment: #26
			{Replicas: 1, Resources: Resources{Requests: Resource{MEM: 30}, Limits: Resource{MEM: 60}}, Value: 10000},                        // existing deployment: #37
			{Replicas: 1, Resources: Resources{Requests: Resource{MEM: 4}, Limits: Resource{MEM: 8}}, Value: AverageRepositoriesRange.Min},   // default
		},
	},
	// CPU usage is based on the number of users it serves (and the size of the index, but we do not account for
	// that here and instead assume a correlation between # users and # repos which is generally true.)
	{
		ServiceName:       "indexedSearchIndexer",
		DockerServiceName: "zoekt-webserver-0",
		ServiceLabel:      "zoekt-webserver",
		PodName:           "indexed-search",
		ScalingFactor:     ByEngagedUsers,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{CPU: 8}, Limits: Resource{CPU: 16}}, Value: UsersRange.Max}, // estimate
			{Resources: Resources{Requests: Resource{CPU: 6}, Limits: Resource{CPU: 12}}, Value: 15000},          // existing deployment: #51
			{Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: 2100},            // existing deployment: #44
			{Resources: Resources{Requests: Resource{CPU: .5}, Limits: Resource{CPU: 2}}, Value: UsersRange.Min}, // default
		},
	},
}

// pods list services which live in the same pod. This is used to ensure we
// recommend the same number of replicas.
var pods = map[string][]string{
	"indexed-search": {"indexedSearch", "indexedSearchIndexer"},
}

var defaults = map[string]map[string]Service{
	"frontend": {
		"kubernetes":     Service{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 8}}},
	},
	"gitserver": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 8, MEM: 8}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 8}}},
	},
	"pgsql": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200},
	},
	"preciseCodeIntel": {
		"kubernetes":     Service{Replicas: 2, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 2, MEM: 4}}},
	},
	"minio": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: .5}, Limits: Resource{CPU: 1, MEM: .5}}, Storage: 100},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: .5}}, Storage: 100},
	},
	"redisCache": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 1}, Limits: Resource{CPU: 7, MEM: 7}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 7}}},
	},
	"redisStore": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 1}, Limits: Resource{CPU: 7, MEM: 7}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 7}}},
	},
	"repoUpdater": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: .5}, Limits: Resource{CPU: 1, MEM: 2}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 4}}},
	},
	"searcher": {
		"kubernetes":     Service{Replicas: 2, Resources: Resources{Requests: Resource{CPU: .5, MEM: .5, EPH: 25}, Limits: Resource{CPU: 2, MEM: 2, EPH: 26}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 2, MEM: 2, EPH: 128}}},
	},
	"symbols": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: .5, EPH: 10}, Limits: Resource{CPU: 2, MEM: 2, EPH: 12}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 2, MEM: 4, EPH: 128}}},
	},
	"syntectServer": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 5, MEM: 6}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 6}}},
	},
	"worker": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 4}}},
	},
	"indexedSearchIndexer": { // zoekt-webserver
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 8, MEM: 8}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 8, MEM: 16}}, Storage: 200},
	},
	"indexedSearch": { // zoekt-indexserver
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 8, MEM: 50}}, Storage: 200},
	},
}
