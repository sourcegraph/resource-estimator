package scaling

// We are using the data gathered from different existing deployments as references for the estimates:
// https://docs.google.com/spreadsheets/d/1N7X_OXDwKk0QSR2Ghbj7ZhjVrQXcMNj-yC8mF1amBi4/edit?usp=sharing

var References = []ServiceScale{
	// Frontend scales based on the number of engaged users.
	// Add 2000 users to user count if code-insight is enabled
	{
		ServiceName:   "frontend",
		ScalingFactor: ByEngagedUsers, // UsersRange = {5, 10000}
		ReferencePoints: []ReferencePoint{
			{Replicas: 5, CPU: Resource{2, 4}, MemoryGB: Resource{8, 16}, Value: UsersRange.Max}, // estimate
			{Replicas: 3, CPU: Resource{4, 8}, MemoryGB: Resource{8, 16}, Value: 5000},           // estimate
			{Replicas: 3, CPU: Resource{2, 4}, MemoryGB: Resource{4, 8}, Value: 2100},            // existing deployment: #4
			{Replicas: 2, CPU: Resource{2, 4}, MemoryGB: Resource{4, 8}, Value: 2050},            // existing deployment: #45
			{Replicas: 2, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}, Value: UsersRange.Min},  // default for instance with <2000 users without code-insight
		},
	},

	// Gitserver scales based on the total size of all repoes and number of average repositories.
	{
		ServiceName:   "gitserver",
		ScalingFactor: ByUserRepoSumRatio,
		ReferencePoints: []ReferencePoint{
			{Replicas: 5, CPU: Resource{16, 16}, MemoryGB: Resource{32, 32}, Value: UserRepoSumRatioRange.Max}, // estimate
			{Replicas: 4, CPU: Resource{16, 16}, MemoryGB: Resource{32, 32}, Value: 150},                       // estimate
			{Replicas: 4, CPU: Resource{8, 8}, MemoryGB: Resource{16, 16}, Value: 50},                          // existing deployment: dogfood
			{Replicas: 3, CPU: Resource{8, 8}, MemoryGB: Resource{32, 32}, Value: 30},                          // estimate
			{Replicas: 3, CPU: Resource{8, 8}, MemoryGB: Resource{16, 16}, Value: 20},                          // estimate
			{Replicas: 2, CPU: Resource{8, 8}, MemoryGB: Resource{32, 32}, Value: 10},                          // estimate
			{Replicas: 2, CPU: Resource{8, 8}, MemoryGB: Resource{16, 16}, Value: 5},                           // estimate
			{Replicas: 1, CPU: Resource{8, 8}, MemoryGB: Resource{16, 16}, Value: 2},                           // estimate
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{8, 8}, Value: UserRepoSumRatioRange.Min},     // default for instance with <4000 repos
		},
	},

	// Memory usage depends on the number of active users and service-connections
	{
		ServiceName:   "pgsql",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 1, CPU: Resource{7, 7}, MemoryGB: Resource{32, 32}, Value: AverageRepositoriesRange.Max}, // existing deployment: dogfood
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{16, 16}, Value: 25000},                        // existing deployment: #4
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{8, 8}, Value: 4000},                           // existing deployment: #43
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{4, 4}, Value: AverageRepositoriesRange.Min},   // bare minimum
		},
	},

	// The entire index must be read into memory to be correlated.
	// Scale vertically when the uploaded index is too large to be processed without OOMing the worker.
	// Scale horizontally to process a higher throughput of indexes.
	// calculation: ~2 times of the size of the largest index
	{
		ServiceName:   "precise-code-intel",
		ScalingFactor: ByLargestIndexSize,
		ReferencePoints: []ReferencePoint{
			{Replicas: 4, CPU: Resource{4, 4}, MemoryGB: Resource{25, 50}, Value: LargestIndexSizeRange.Max}, // calculation
			{Replicas: 4, CPU: Resource{4, 4}, MemoryGB: Resource{20, 41}, Value: 81},                        // calculation
			{Replicas: 3, CPU: Resource{4, 4}, MemoryGB: Resource{29, 58}, Value: 80},                        // calculation
			{Replicas: 3, CPU: Resource{4, 4}, MemoryGB: Resource{20, 40}, Value: 61},                        // calculation
			{Replicas: 2, CPU: Resource{4, 4}, MemoryGB: Resource{30, 60}, Value: 60},                        // calculation
			{Replicas: 2, CPU: Resource{4, 4}, MemoryGB: Resource{16, 32}, Value: 32},                        // calculation
			{Replicas: 2, CPU: Resource{4, 4}, MemoryGB: Resource{4, 8}, Value: 8},                           // calculation
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{8, 16}, Value: 7},                          // calculation
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{2, 4}, Value: LargestIndexSizeRange.Min},   // bare minimum
		},
	},

	// Searcher replicas scale based the number of concurrent unidexed queries & number concurrent of structural searches
	{
		ServiceName:   "searcher",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 8, Value: AverageRepositoriesRange.Max}, // estimate
			{Replicas: 6, Value: 25000},                        // existing deployment: #4 & 12
			{Replicas: 4, Value: 14000},                        // existing deployment: #51
			{Replicas: 1, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},
	// Searcher is IO and CPU bound. It fetches archives from gitserver and searches them with regexp.
	// Memory scales based on the size of repositories (i.e. when large monorepos are in the picture).
	{
		ServiceName:   "searcher",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{CPU: Resource{3, 6}, MemoryGB: Resource{4, 8}, Value: AverageRepositoriesRange.Max},   // estimate
			{CPU: Resource{3, 6}, MemoryGB: Resource{4, 8}, Value: 25000},                          // existing deployment: #4
			{CPU: Resource{.5, 2}, MemoryGB: Resource{2, 4}, Value: 4000},                          // existing deployment: #47
			{CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},

	// Symbols replicas scale based on the number of average repositories, and its resources scale
	// based on the size of repositories (i.e. when large monorepos are in the picture).
	{
		ServiceName:   "symbols",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 4, Value: AverageRepositoriesRange.Max}, // estimate
			{Replicas: 3, Value: 25000},                        // estimate
			{Replicas: 2, Value: 4000},                         // estimate
			{Replicas: 1, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},
	{
		ServiceName:   "symbols",
		ScalingFactor: ByLargeMonorepos,
		ReferencePoints: []ReferencePoint{
			{CPU: Resource{2, 4}, MemoryGB: Resource{8, 16}, Value: LargeMonoreposRange.Max},  // estimate
			{CPU: Resource{2, 4}, MemoryGB: Resource{2, 8}, Value: 2},                         // estimate
			{CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}, Value: LargeMonoreposRange.Min}, // bare minimum
		},
	},

	// At initialization time, many highlighting themes and compiled grammars are loaded into memory.
	// There is additional memory consumption on receiving requests (< 25 MB), although,
	// that's generally much smaller than the constant overhead (1-2 GB).
	// In some situations, there are hangs with syntax highlighting.
	// These can cause runaway CPU usage (for 1 core per hang).
	// syntect-server should normally kill such processes and restart them if that happens.
	{
		ServiceName:   "syntect-server",
		ScalingFactor: ByEngagedUsers,
		ReferencePoints: []ReferencePoint{
			{Replicas: 1, CPU: Resource{2, 8}, MemoryGB: Resource{6, 12}, Value: UsersRange.Max}, // estimate
			{Replicas: 1, CPU: Resource{.5, 4}, MemoryGB: Resource{2, 6}, Value: 5000},           // existing deployment: average between 27 and
			{Replicas: 1, CPU: Resource{.5, 4}, MemoryGB: Resource{2, 6}, Value: UsersRange.Min}, // bare minimum
		},
	},

	// worker is used by different services, and mostly scale based on the number of average repositories to execute jobs
	{
		ServiceName:   "worker",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{4, 16}, Value: AverageRepositoriesRange.Max}, // estimate
			{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{4, 8}, Value: 4000},                          // estimate
			{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{2, 4}, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},

	// zoekt-indexserver memory usage scales based on whether it must index large monorepos
	{
		ServiceName:   "zoekt-indexserver",
		ScalingFactor: ByLargeMonorepos,
		ReferencePoints: []ReferencePoint{
			{MemoryGB: Resource{16, 16}, Value: LargeMonoreposRange.Max}, // estimate
			{MemoryGB: Resource{16, 16}, Value: 2},                       // estimate
			{MemoryGB: Resource{8, 8}, Value: LargeMonoreposRange.Min},   // bare minimum
		},
	},
	// CPU usage and replicas scale based on the number of average repos it must index as it indexes one repo at a time
	// Set replica number to 0 as it will be synced with the replica number for webserver
	{
		ServiceName:   "zoekt-indexserver",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 4, CPU: Resource{4, 8}, Value: AverageRepositoriesRange.Max}, // estimate: at 50k repos the instance will have 4 replics so 8CPU limit per replica should be enough
			{Replicas: 2, CPU: Resource{4, 8}, Value: 14000},                        // existing deployment: #26 - 16 CPU / 2 replicas = 8
			{Replicas: 1, CPU: Resource{4, 8}, Value: 10000},                        // existing deployment: #37
			{Replicas: 1, CPU: Resource{4, 8}, Value: 1500},                         // existing deployment: #44
			{Replicas: 1, CPU: Resource{4, 8}, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},

	// zoekt-webserver memory usage and replicas scale based on how many average repositories it is
	// serving (roughly 2/3 the size of the actual repos is the memory usage).
	{
		ServiceName:   "zoekt-webserver",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 4, MemoryGB: Resource{25, 50}, Value: AverageRepositoriesRange.Max}, // existing deployment: dogfood
			{Replicas: 2, MemoryGB: Resource{8, 16}, Value: 14000},                         // existing deployment: #26
			{Replicas: 1, MemoryGB: Resource{30, 60}, Value: 10000},                        // existing deployment: #37
			{Replicas: 1, MemoryGB: Resource{4, 8}, Value: AverageRepositoriesRange.Min},   // bare minimum
		},
	},
	// CPU usage is based on the number of users it serves (and the size of the index, but we do not account for
	// that here and instead assume a correlation between # users and # repos which is generally true.)
	{
		ServiceName:   "zoekt-webserver",
		ScalingFactor: ByEngagedUsers,
		ReferencePoints: []ReferencePoint{
			{CPU: Resource{8, 16}, Value: UsersRange.Max}, // estimate
			{CPU: Resource{6, 12}, Value: 15000},          // existing deployment: #51
			{CPU: Resource{4, 8}, Value: 2100},            // existing deployment: #44
			{CPU: Resource{.5, 2}, Value: UsersRange.Min}, // bare minimum
		},
	},
}

// pods list services which live in the same pod. This is used to ensure we
// recommend the same number of replicas.
var pods = map[string][]string{
	"indexed-search": {"zoekt-webserver", "zoekt-indexserver"},
}

var defaults = map[string]map[string]ReferencePoint{
	"kubernetes": {
		"frontend":           ReferencePoint{Replicas: 2, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}},
		"gitserver":          ReferencePoint{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{8, 8}},
		"pgsql":              ReferencePoint{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{4, 4}},
		"precise-code-intel": ReferencePoint{Replicas: 2, CPU: Resource{.5, 2}, MemoryGB: Resource{2, 4}},
		"redis-store":        ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{7, 7}},
		"redis-cache":        ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{7, 7}},
		"repo-updater":       ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{.5, 2}},
		"searcher":           ReferencePoint{Replicas: 2, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}},
		"symbols":            ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}},
		"syntect-server":     ReferencePoint{Replicas: 1, CPU: Resource{.5, 4}, MemoryGB: Resource{2, 6}},
		"worker":             ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{2, 4}},
		"zoekt-webserver":    ReferencePoint{Replicas: 1, CPU: Resource{4, 8}, MemoryGB: Resource{4, 8}},
		"zoekt-indexserver":  ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{2, 4}},
	},
	"docker-compose": {
		"frontend":           ReferencePoint{Replicas: 2, CPU: Resource{4, 4}, MemoryGB: Resource{8, 8}},
		"gitserver":          ReferencePoint{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{8, 8}},
		"pgsql":              ReferencePoint{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{4, 4}},
		"precise-code-intel": ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{2, 4}},
		"redis-store":        ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{7, 7}},
		"redis-cache":        ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{7, 7}},
		"repo-updater":       ReferencePoint{Replicas: 1, CPU: Resource{.1, 4}, MemoryGB: Resource{.5, 4}},
		"searcher":           ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}},
		"symbols":            ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 4}},
		"syntect-server":     ReferencePoint{Replicas: 1, CPU: Resource{.25, 4}, MemoryGB: Resource{2, 6}},
		"worker":             ReferencePoint{Replicas: 1, CPU: Resource{4, 4}, MemoryGB: Resource{4, 4}},
		"zoekt-webserver":    ReferencePoint{Replicas: 1, CPU: Resource{8, 8}, MemoryGB: Resource{16, 16}},
		"zoekt-indexserver":  ReferencePoint{Replicas: 1, CPU: Resource{8, 8}, MemoryGB: Resource{50, 50}},
	},
}
