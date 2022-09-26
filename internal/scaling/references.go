package scaling

// We are using the data gathered from different existing deployments as references for the estimates:
// https://docs.google.com/spreadsheets/d/1N7X_OXDwKk0QSR2Ghbj7ZhjVrQXcMNj-yC8mF1amBi4/edit?usp=sharing
// TODO: UPDATE DATA REFERENCE LINK AND DISPLAY NEW & MISSING SERVICES

var References = []ServiceScale{
	{
		ServiceName:       "frontend",
		ServiceLabel:      "sourcegraph-frontend",
		DockerServiceName: "sourcegraph-frontend-0",
		PodName:           "frontend",
		ScalingFactor:     ByEngagedUsers,
		ReferencePoints: []Service{
			{Replicas: 4, Resources: Resources{Requests: Resource{CPU: 8, MEM: 48}, Limits: Resource{CPU: 8, MEM: 48}}, Value: UsersRange.Max}, // Cloud
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 8, MEM: 24}, Limits: Resource{CPU: 8, MEM: 24}}, Value: 25000},          // XL
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 4, MEM: 3}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 10000},            // L
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 4, MEM: 3}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 5000},             // M
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: 1000},             // S
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: UsersRange.Min},   // default
		},
	},
	{
		ServiceName:       "gitserver",
		ServiceLabel:      "gitserver",
		DockerServiceName: "gitserver-0",
		PodName:           "gitserver",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 10, Resources: Resources{Requests: Resource{CPU: 30}, Limits: Resource{CPU: 30}}, Value: AverageRepositoriesRange.Max}, // Cloud
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 6}, Limits: Resource{CPU: 12}}, Value: 250000},                         // XL
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: 100000},                          // L
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 3}, Limits: Resource{CPU: 6}}, Value: 5000},                            // M
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2}, Limits: Resource{CPU: 4}}, Value: 1000},                            // S
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2}, Limits: Resource{CPU: 4}}, Value: AverageRepositoriesRange.Min},    // default
		},
	},
	{
		ServiceName:       "gitserver",
		ServiceLabel:      "gitserver",
		DockerServiceName: "gitserver-0",
		PodName:           "gitserver",
		ScalingFactor:     ByTotalRepoSize,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{MEM: 2500000}, Limits: Resource{MEM: 2500000}}, Value: TotalRepoSizeRange.Max},
			{Resources: Resources{Requests: Resource{MEM: 50000}, Limits: Resource{MEM: 50000}}, Value: 1000000},
			{Resources: Resources{Requests: Resource{MEM: 5000}, Limits: Resource{MEM: 5000}}, Value: 100000},
			{Resources: Resources{Requests: Resource{MEM: 500}, Limits: Resource{MEM: 500}}, Value: 10000},
			{Resources: Resources{Requests: Resource{MEM: 50}, Limits: Resource{MEM: 50}}, Value: 1000},
			{Resources: Resources{Requests: Resource{MEM: 5}, Limits: Resource{MEM: 5}}, Value: 100},
			{Resources: Resources{Requests: Resource{MEM: 4}, Limits: Resource{MEM: 4}}, Value: TotalRepoSizeRange.Min}, // default
		},
	},
	{
		ServiceName:       "minio",
		ServiceLabel:      "minio",
		DockerServiceName: "minio",
		PodName:           "minio",
		ScalingFactor:     ByLargestIndexSize,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 1}, Limits: Resource{CPU: 1, MEM: 1}}, Storage: 1000, Value: LargestIndexSizeRange.Max}, // calculation
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: .5}, Limits: Resource{CPU: 1, MEM: .5}}, Storage: 1, Value: LargestIndexSizeRange.Min},  // bare minimum
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
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 12, MEM: 36}, Limits: Resource{CPU: 12, MEM: 36}}, Storage: 200, Value: AverageRepositoriesRange.Max}, // Estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 8, MEM: 32}, Limits: Resource{CPU: 8, MEM: 32}}, Storage: 200, Value: 250000},                         // XL
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 6}, Limits: Resource{CPU: 4, MEM: 6}}, Storage: 200, Value: 10000},                            // L
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200, Value: 500},                              // L
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200, Value: AverageRepositoriesRange.Min},     // default
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

	{
		ServiceName:       "redisCache",
		ServiceLabel:      "redis-cache",
		DockerServiceName: "redis-cache",
		PodName:           "redis",
		ScalingFactor:     ByUserRepoSumRatio,
		ReferencePoints: []Service{
			{Replicas: 4, Resources: Resources{Requests: Resource{CPU: 1, MEM: 7}, Limits: Resource{CPU: 1, MEM: 7}}, Storage: 100, Value: UserRepoSumRatioRange.Max}, // estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 1}, Limits: Resource{CPU: 1, MEM: 1}}, Storage: 100, Value: UserRepoSumRatioRange.Min}, // bare minimum
		},
	},
	{
		ServiceName:       "redisStore",
		ServiceLabel:      "redis-store",
		DockerServiceName: "redis-store",
		PodName:           "redis",
		ScalingFactor:     ByEngagedUsers,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 1}, Limits: Resource{CPU: 1, MEM: 7}}, Storage: 100, Value: UsersRange.Max},  // estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 1}, Limits: Resource{CPU: 1, MEM: 1}}, Storage: 100, Value: UsersRange.Min}, // bare minimum
		},
	},

	// Searcher replicas scale based the number of concurrent unidexed queries & number concurrent of structural searches
	// Searcher is IO and CPU bound. It fetches archives from gitserver and searches them with regexp.
	// Memory scales based on the size of repositories (i.e. when large monorepos are in the picture).
	// Formula: replica for every 500k repos
	// Formula for CPU - Add 2 CPU for every size up / number of replica
	// Formula for MEM - Add 4 MEM for every size up / number of replica
	{
		ServiceName:       "searcher",
		ServiceLabel:      "searcher",
		DockerServiceName: "searcher-0",
		PodName:           "searcher",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 10, Resources: Resources{Requests: Resource{CPU: 4, MEM: 8}, Limits: Resource{CPU: 6, MEM: 8}}, Value: AverageRepositoriesRange.Max}, // Cloud
			{Replicas: 6, Resources: Resources{Requests: Resource{CPU: 3, MEM: 8}, Limits: Resource{CPU: 6, MEM: 8}}, Value: 4000000},
			{Replicas: 5, Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Value: 2500000},
			{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 4, MEM: 12}, Limits: Resource{CPU: 8, MEM: 12}}, Value: 1000000},
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 6, MEM: 20}, Limits: Resource{CPU: 12, MEM: 20}}, Value: 500000},                      // Estimate
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 5, MEM: 16}, Limits: Resource{CPU: 10, MEM: 16}}, Value: 250000},                      // Size XL
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 12}, Limits: Resource{CPU: 8, MEM: 12}}, Value: 100000},                       // Size L
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 3, MEM: 8}, Limits: Resource{CPU: 6, MEM: 8}}, Value: 50000},                          // Size M
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Value: 1000},                           // Size S
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: .5}, Limits: Resource{CPU: 2, MEM: 2}}, Value: AverageRepositoriesRange.Min}, // default
		},
	},

	// Symbols replicas scale based on the number of average repositories, and its resources scale
	// based on the size of repositories (i.e. when large monorepos are in the picture).
	// Formula: Replica for every 1million repos
	{
		ServiceName:       "symbols",
		ServiceLabel:      "symbols",
		DockerServiceName: "symbols-0",
		PodName:           "symbols",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Replicas: 5, Value: AverageRepositoriesRange.Max}, // Cloud
			{Replicas: 5, Value: 4000000},
			{Replicas: 4, Value: 3000000},
			{Replicas: 3, Value: 2000000},
			{Replicas: 2, Value: 1000000},
			{Replicas: 1, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},
	{
		ServiceName:       "symbols",
		ServiceLabel:      "symbols",
		DockerServiceName: "symbols-0",
		PodName:           "symbols",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 16, MEM: 64}}, Value: AverageRepositoriesRange.Max}, // Cloud
			{Resources: Resources{Requests: Resource{CPU: 4, MEM: 16}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 250000},                        // Size XL
			{Resources: Resources{Requests: Resource{CPU: 4, MEM: 12}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 100000},                        // Size L
			{Resources: Resources{Requests: Resource{CPU: 3, MEM: 8}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 50000},                          // Size M
			{Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: 1000},
			{Resources: Resources{Requests: Resource{CPU: .5, MEM: .5}, Limits: Resource{CPU: 2, MEM: 4}}, Value: AverageRepositoriesRange.Min}, // default
		},
	},
	// TODO
	{
		ServiceName:       "symbols",
		ServiceLabel:      "symbols",
		DockerServiceName: "symbols-0",
		PodName:           "symbols",
		ScalingFactor:     ByLargestRepoSize,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{EPH: 50000000}, Limits: Resource{EPH: 60000000}}, Value: LargestRepoSizeRange.Max}, // calculation
			{Resources: Resources{Requests: Resource{EPH: 50000}, Limits: Resource{EPH: 60000}}, Value: 50000},                          // calculation
			{Resources: Resources{Requests: Resource{EPH: 100}, Limits: Resource{EPH: 120}}, Value: 100},                                // calculation
			{Resources: Resources{Requests: Resource{EPH: 50}, Limits: Resource{EPH: 60}}, Value: 50},                                   // calculation
			{Resources: Resources{Requests: Resource{EPH: 5}, Limits: Resource{EPH: 6}}, Value: 5},                                      // calculation
			{Resources: Resources{Requests: Resource{EPH: 2}, Limits: Resource{EPH: 3}}, Value: 2},                                      // calculation
			{Resources: Resources{Requests: Resource{EPH: 1}, Limits: Resource{EPH: 2}}, Value: LargestRepoSizeRange.Min},               // bare minimum
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
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 8}, Limits: Resource{CPU: 16, MEM: 18}}, Value: UsersRange.Max}, // Cloud
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 3}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 500000},
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 2}, Limits: Resource{CPU: 4, MEM: 6}}, Value: 25000},
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .25, MEM: 2}, Limits: Resource{CPU: 4, MEM: 6}}, Value: UsersRange.Min}, // default
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
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 4}, Limits: Resource{CPU: 4, MEM: 8}}, Value: AverageRepositoriesRange.Max},  // Cloud
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}, Value: AverageRepositoriesRange.Min}, // default
		},
	},

	// zoekt-indexserver memory usage scales based on whether it must index large monorepos
	{
		ServiceName:       "indexedSearch",
		ServiceLabel:      "zoekt-indexserver",
		DockerServiceName: "zoekt-indexserver-0",
		PodName:           "indexed-search",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 10}}, Value: AverageRepositoriesRange.Max}, // Cloud
			{Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 10}}, Value: 500000},                       // Size XL
			{Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 10}}, Value: 250000},                       // Size L
			{Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 10}}, Value: 100000},                       // Size M
			{Resources: Resources{Requests: Resource{MEM: 4}, Limits: Resource{MEM: 10}}, Value: 5000},                         // Size S
			{Resources: Resources{Requests: Resource{MEM: 4}, Limits: Resource{MEM: 8}}, Value: AverageRepositoriesRange.Min},  // default
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
			{Replicas: 0, Resources: Resources{Requests: Resource{CPU: 8}, Limits: Resource{CPU: 10}}, Value: AverageRepositoriesRange.Max}, // Cloud
			{Replicas: 0, Resources: Resources{Requests: Resource{CPU: 5}, Limits: Resource{CPU: 10}}, Value: 500000},                       // Size XL
			{Replicas: 0, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 10}}, Value: 250000},                       // Size L
			{Replicas: 0, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 10}}, Value: 100000},                       // Size M
			{Replicas: 0, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: 5000},                          // Size S
			{Replicas: 0, Resources: Resources{Requests: Resource{CPU: 4}, Limits: Resource{CPU: 8}}, Value: AverageRepositoriesRange.Min},  // default / Size XS
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
			{Replicas: 50, Resources: Resources{Requests: Resource{MEM: 2160}, Limits: Resource{MEM: 4320}}, Value: AverageRepositoriesRange.Max}, // Cloud
			{Replicas: 5, Resources: Resources{Requests: Resource{MEM: 24}, Limits: Resource{MEM: 48}}, Value: 500000},                            // Size XL
			{Replicas: 3, Resources: Resources{Requests: Resource{MEM: 16}, Limits: Resource{MEM: 32}}, Value: 250000},                            // Size L
			{Replicas: 2, Resources: Resources{Requests: Resource{MEM: 8}, Limits: Resource{MEM: 16}}, Value: 100000},                             // Size M
			{Replicas: 1, Resources: Resources{Requests: Resource{MEM: 4}, Limits: Resource{MEM: 8}}, Value: 5000},                                // Size S
			{Replicas: 1, Resources: Resources{Requests: Resource{MEM: 2}, Limits: Resource{MEM: 4}}, Value: AverageRepositoriesRange.Min},        // default / Size XS
		},
	},
	// CPU usage is based on the number of users it serves (and the size of the index, but we do not account for
	// that here and instead assume a correlation between # users and # repos which is generally true.)
	{
		ServiceName:       "indexedSearchIndexer",
		DockerServiceName: "zoekt-webserver-0",
		ServiceLabel:      "zoekt-webserver",
		PodName:           "indexed-search",
		ScalingFactor:     ByAverageRepositories,
		ReferencePoints: []Service{
			{Resources: Resources{Requests: Resource{CPU: 8}, Limits: Resource{CPU: 384}}, Value: AverageRepositoriesRange.Max}, // Cloud
			{Resources: Resources{Requests: Resource{CPU: 18}, Limits: Resource{CPU: 36}}, Value: 500000},                       // Size XL
			{Resources: Resources{Requests: Resource{CPU: 8}, Limits: Resource{CPU: 16}}, Value: 250000},                        // Size L
			{Resources: Resources{Requests: Resource{CPU: 3}, Limits: Resource{CPU: 6}}, Value: 100000},                         // Size M
			{Resources: Resources{Requests: Resource{CPU: 2}, Limits: Resource{CPU: 4}}, Value: 5000},                           // Size S
			{Resources: Resources{Requests: Resource{CPU: .5}, Limits: Resource{CPU: 2}}, Value: AverageRepositoriesRange.Min},  // default / Size XS
		},
	},
	{
		ServiceName:       "codeinsights-db",
		DockerServiceName: "codeinsights-db",
		ServiceLabel:      "codeinsights-db",
		PodName:           "codeinsights-db",
		ScalingFactor:     ByLargestIndexSize,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200, Value: LargestIndexSizeRange.Max}, // Enabled
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200, Value: LargestIndexSizeRange.Min}, // Disabled
		},
	},
	{
		ServiceName:       "codeintel-db",
		DockerServiceName: "codeintel-db",
		ServiceLabel:      "codeintel-db",
		PodName:           "codeintel-db",
		ScalingFactor:     ByLargestIndexSize,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200, Value: LargestIndexSizeRange.Max}, // Enabled
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200, Value: LargestIndexSizeRange.Min}, // Disabled
		},
	},
	// Use default values
	{
		ServiceName:       "prometheus",
		DockerServiceName: "prometheus",
		ServiceLabel:      "prometheus",
		PodName:           "prometheus",
		ScalingFactor:     ByLargestIndexSize,
		ReferencePoints: []Service{
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 6}, Limits: Resource{CPU: 2, MEM: 6}}, Storage: 200, Value: LargestIndexSizeRange.Max}, // Enabled
			{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 6}, Limits: Resource{CPU: 2, MEM: 6}}, Storage: 200, Value: LargestIndexSizeRange.Min}, // Disabled
		},
	},
}

// pods list services which live in the same pod. This is used to ensure we
// recommend the same number of replicas.
var pods = map[string][]string{
	"indexed-search": {"indexedSearch", "indexedSearchIndexer"},
}

var defaults = map[string]map[string]Service{
	"cadvisor": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .15, MEM: .2}, Limits: Resource{CPU: .3, MEM: .2}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 1}}},
	},
	"codeinsights-db": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 2, MEM: 4}}, Storage: 128},
	},
	"codeintel-db": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 4}}, Storage: 128},
	},
	"frontend": {
		"kubernetes":     Service{Replicas: 2, Resources: Resources{Requests: Resource{CPU: 2, MEM: 2, EPH: 4}, Limits: Resource{CPU: 2, MEM: 4, EPH: 8}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 8}}, Storage: 128},
	},
	"frontend-internal": {
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 8}}, Storage: 128},
	},
	"github-proxy": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .1, MEM: .25}, Limits: Resource{CPU: 1, MEM: 1}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 1}}},
	},
	"gitserver": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 8}, Limits: Resource{CPU: 4, MEM: 8}}, Storage: 200},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 8}}, Storage: 200},
	},
	"grafana": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .1, MEM: .512}, Limits: Resource{CPU: 1, MEM: .512}}, Storage: 2},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 1}}, Storage: 2},
	},
	"jaeger": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: .5}, Limits: Resource{CPU: 1, MEM: 1}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: .5, MEM: .512}}},
	},
	"otel-collector": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 1}, Limits: Resource{CPU: 2, MEM: 3}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 1}}},
	},
	"pgsql": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 4, MEM: 4}}, Storage: 200},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 4}}, Storage: 128},
	},
	"preciseCodeIntel": {
		"kubernetes":     Service{Replicas: 2, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 2, MEM: 4}}},
	},
	"prometheus": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 6}, Limits: Resource{CPU: 2, MEM: 6}}, Storage: 200},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 8}}, Storage: 200},
	},
	"minio": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: .5}, Limits: Resource{CPU: 1, MEM: .5}}, Storage: 100},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 1}}, Storage: 128},
	},
	"redisCache": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 7}, Limits: Resource{CPU: 1, MEM: 7}}, Storage: 100},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 7}}, Storage: 128},
	},
	"redisStore": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: 7}, Limits: Resource{CPU: 1, MEM: 7}}, Storage: 100},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 1, MEM: 7}}, Storage: 128},
	},
	"repoUpdater": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 1, MEM: .5}, Limits: Resource{CPU: 1, MEM: 2}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 4}}, Storage: 128},
	},
	"searcher": {
		"kubernetes":     Service{Replicas: 2, Resources: Resources{Requests: Resource{CPU: .5, MEM: .5, EPH: 25}, Limits: Resource{CPU: 2, MEM: 2, EPH: 26}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 2, MEM: 2}}, Storage: 128},
	},
	"symbols": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: .5, EPH: 10}, Limits: Resource{CPU: 2, MEM: 2, EPH: 12}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 2, MEM: 4}}, Storage: 128},
	},
	"syntectServer": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .25, MEM: 2}, Limits: Resource{CPU: 4, MEM: 6}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 6}}}, // no disk
	},
	"worker": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 4, MEM: 4}}, Storage: 128},
	},
	// zoekt-webserver
	"indexedSearchIndexer": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: 4, MEM: 4}, Limits: Resource{CPU: 8, MEM: 8}}, Storage: 200},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 8, MEM: 16}}, Storage: 200},
	},
	// zoekt-indexserver
	"indexedSearch": {
		"kubernetes":     Service{Replicas: 1, Resources: Resources{Requests: Resource{CPU: .5, MEM: 2}, Limits: Resource{CPU: 2, MEM: 4}}},
		"docker-compose": Service{Replicas: 1, Resources: Resources{Limits: Resource{CPU: 8, MEM: 50}}, Storage: 200},
	},
}
