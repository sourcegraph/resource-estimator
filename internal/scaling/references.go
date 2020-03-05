package scaling

// Spreadsheet of known good deployment configurations:
// https://docs.google.com/spreadsheets/d/1in1sfEkgXGVB2_HInX93bxNOFJPA_r3ugfD5lEKCk_U/edit#gid=0
var References = []ServiceScale{
	// Frontend scales based on the number of engaged users.
	{
		ServiceName:   "frontend",
		ScalingFactor: ByEngagedUsers,
		ReferencePoints: []ReferencePoint{
			{Replicas: 9, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}, Value: UsersRange.Max},    // projection
			{Replicas: 5, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}, Value: 1750 + (1425 * 2)}, // projection
			{Replicas: 4, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}, Value: 1750 + (1425 * 1)}, // projection
			{Replicas: 3, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}, Value: 7000 * .25},        // 1750 users -- row 3 of spreadsheet
			{Replicas: 3, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}, Value: 1300 * .25},        // 325 users -- row 2 of spreadsheet
			{Replicas: 1, CPU: Resource{2, 2}, MemoryGB: Resource{2, 4}, Value: UsersRange.Min},    // bare minimum
		},
	},

	// Symbols replicas scale based on the number of average repositories, and its resources scale
	// based on the size of repositories (i.e. when large monorepos are in the picture).
	{
		ServiceName:   "symbols",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 8, Value: AverageRepositoriesRange.Max}, // projection
			{Replicas: 6, Value: 15000 + 13500},                // 28500 repos -- projection
			{Replicas: 4, Value: 15000},                        // row 3 of spreadsheet
			{Replicas: 2, Value: 1500},                         // row 4 of spreadsheet
			{Replicas: 1, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},
	{
		ServiceName:   "symbols",
		ScalingFactor: ByLargeMonorepos,
		ReferencePoints: []ReferencePoint{
			{CPU: Resource{2, 4}, MemoryGB: Resource{1, 4}, Value: LargeMonoreposRange.Max},   // estimate based on entire spreadsheet
			{CPU: Resource{2, 4}, MemoryGB: Resource{1, 4}, Value: 1},                         // estimate based on entire spreadsheet
			{CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}, Value: LargeMonoreposRange.Min}, // bare minimum
		},
	},

	// Gitserver scales based on the number of average repositories.
	{
		ServiceName:   "gitserver",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 5, CPU: Resource{4, 8}, MemoryGB: Resource{4, 8}, Value: AverageRepositoriesRange.Max}, // projection
			{Replicas: 4, CPU: Resource{4, 8}, MemoryGB: Resource{4, 8}, Value: 15000 + 13500},                // projection
			{Replicas: 3, CPU: Resource{4, 8}, MemoryGB: Resource{4, 8}, Value: 15000},                        // from row 3 of spreadsheet
			{Replicas: 2, CPU: Resource{4, 8}, MemoryGB: Resource{4, 8}, Value: 1500},                         // from row 4 of spreadsheet
			{Replicas: 1, CPU: Resource{4, 8}, MemoryGB: Resource{4, 8}, Value: AverageRepositoriesRange.Min}, // bare minimum
		},
	},

	// Searcher replicas scale based on the number of average repositories, and its resources scale
	// based on the size of repositories (i.e. when large monorepos are in the picture).
	{
		ServiceName:   "searcher",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 12, Value: AverageRepositoriesRange.Max}, // projection
			{Replicas: 9, Value: 15000 + 13500},                 // 28500 repos -- projection
			{Replicas: 6, Value: 15000},                         // row 3 of spreadsheet
			{Replicas: 3, Value: 1500},                          // row 4 of spreadsheet
			{Replicas: 1, Value: AverageRepositoriesRange.Min},  // bare minimum
		},
	},
	{
		ServiceName:   "searcher",
		ScalingFactor: ByLargeMonorepos,
		ReferencePoints: []ReferencePoint{
			{CPU: Resource{.5, 2}, MemoryGB: Resource{1, 4}, Value: LargeMonoreposRange.Max},  // speculative
			{CPU: Resource{.5, 2}, MemoryGB: Resource{1, 4}, Value: 1},                        // speculative
			{CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}, Value: LargeMonoreposRange.Min}, // bare minimum
		},
	},

	// Replacer replicas scale based on the number of average repositories, and its resources scale
	// based on the size of repositories (i.e. when large monorepos are in the picture).
	{
		ServiceName:   "replacer",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 12, Value: AverageRepositoriesRange.Max}, // projection
			{Replicas: 9, Value: 15000 + 13500},                 // 28500 repos -- projection
			{Replicas: 6, Value: 15000},                         // speculative and based on "replacer is nearly identical to searcher in scaling"
			{Replicas: 3, Value: 1500},                          // speculative and based on "replacer is nearly identical to searcher in scaling"
			{Replicas: 1, Value: AverageRepositoriesRange.Min},  // bare minimum
		},
	},
	{
		ServiceName:   "replacer",
		ScalingFactor: ByLargeMonorepos,
		ReferencePoints: []ReferencePoint{
			{CPU: Resource{2, 4}, MemoryGB: Resource{2, 2}, Value: LargeMonoreposRange.Max},    // very speculative
			{CPU: Resource{1, 4}, MemoryGB: Resource{1, 1}, Value: 1},                          // very speculative
			{CPU: Resource{.5, 4}, MemoryGB: Resource{.5, .5}, Value: LargeMonoreposRange.Min}, // bare minimum
		},
	},

	// zoekt-indexserver memory usage scales based on whether it must index large monorepos. Its
	// CPU usage and replicas scale based on the number of average repos it must index.
	{
		ServiceName:   "zoekt-indexserver",
		ScalingFactor: ByLargeMonorepos,
		ReferencePoints: []ReferencePoint{
			{MemoryGB: Resource{16, 16}, Value: LargeMonoreposRange.Max}, // speculative
			{MemoryGB: Resource{16, 16}, Value: 1},                       // from row 9 of spreadsheet
			{MemoryGB: Resource{4, 8}, Value: LargeMonoreposRange.Min},   // bare minimum
		},
	},
	{
		ServiceName:   "zoekt-indexserver",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 2, CPU: Resource{6, 12}, Value: AverageRepositoriesRange.Max}, // speculative based on row 8 of spreadsheet
			{Replicas: 2, CPU: Resource{4, 8}, Value: 17000},                         // derived from row 9 of spreadsheet
			{Replicas: 1, CPU: Resource{4, 8}, Value: AverageRepositoriesRange.Min},  // bare minimum
		},
	},

	// zoekt-webserver memory usage and replicas scale based on how many average repositories it is
	// serving (roughly 2/3 the size of the actual repos is the memory usage). Its CPU usage is
	// based on the number of users it serves (and the size of the index, but we do not account for
	// that here and instead assume a correlation between # users and # repos which is generally
	// true.)
	{
		ServiceName:   "zoekt-webserver",
		ScalingFactor: ByAverageRepositories,
		ReferencePoints: []ReferencePoint{
			{Replicas: 2, MemoryGB: Resource{80, 80}, Value: AverageRepositoriesRange.Max}, // derived from row 8 of spreadsheet
			{Replicas: 2, MemoryGB: Resource{50, 50}, Value: 17000},                        // derived from row 9 of spreadsheet
			{Replicas: 1, MemoryGB: Resource{64, 64}, Value: 11000},                        // derived from row 2 of spreadsheet
			{Replicas: 1, MemoryGB: Resource{34, 34}, Value: 1500},                         // derived from row 4 of spreadsheet
			{Replicas: 1, MemoryGB: Resource{4, 8}, Value: AverageRepositoriesRange.Min},   // bare minimum
		},
	},
	{
		ServiceName:   "zoekt-webserver",
		ScalingFactor: ByEngagedUsers,
		ReferencePoints: []ReferencePoint{
			{CPU: Resource{192, 192}, Value: UsersRange.Max},     // projection
			{CPU: Resource{48, 48}, Value: UsersRange.Max * .25}, // projection
			{CPU: Resource{16, 16}, Value: 210 * 4},              // 840 engaged users -- derived from row 9 of spreadsheet
			{CPU: Resource{12, 12}, Value: 1300 * .50},           // 650 engaged users -- row 2 of spreadsheet
			{CPU: Resource{.5, 2}, Value: UsersRange.Min},        // bare minimum
		},
	},

	// syntect_server internally runs 4 worker processes, each of which can consume up ot 1.1G of
	// memory and concurrently serves many HTTP requests. Once its memory reaches 4.4G total,
	// scaling becomes linear based on request load, primarily with CPU being the bottleneck.
	{
		ServiceName:   "syntect-server",
		ScalingFactor: ByEngagedUsers,
		ReferencePoints: []ReferencePoint{
			{Replicas: 1, CPU: Resource{12, 64}, MemoryGB: Resource{6, 10}, Value: UsersRange.Max}, // speculative
			{Replicas: 1, CPU: Resource{6, 32}, MemoryGB: Resource{5, 9}, Value: 8000},             // speculative
			{Replicas: 1, CPU: Resource{4, 16}, MemoryGB: Resource{4, 8}, Value: 6000},             // speculative
			{Replicas: 1, CPU: Resource{2, 8}, MemoryGB: Resource{3, 7}, Value: 4000},              // speculative
			{Replicas: 1, CPU: Resource{.25, 4}, MemoryGB: Resource{2, 6}, Value: 2000},            // Derived from https://sourcegraph.slack.com/archives/D2QCCPZBP/p1582591335002100
			{Replicas: 1, CPU: Resource{.25, 4}, MemoryGB: Resource{2, 6}, Value: UsersRange.Min},  // bare minimum
		},
	},
}


var defaults = map[string]map[string]ReferencePoint{
	"kubernetes": {
		"prometheus": ReferencePoint{Replicas: 1, CPU: Resource{.5, .5}, MemoryGB: Resource{2, 2}},
		"query-runner": ReferencePoint{Replicas: 1, CPU: Resource{.5, 1}, MemoryGB: Resource{1, 1}},
		"redis-store": ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{6, 6}},
		"redis-cache": ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{6, 6}},
		"replacer": ReferencePoint{Replicas: 1, CPU: Resource{.5, 4}, MemoryGB: Resource{.5, .5}},
		"repo-updater": ReferencePoint{Replicas: 1, CPU: Resource{.1, .1}, MemoryGB: Resource{.5, .5}},
		"searcher": ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}},
		"symbols": ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}},
		"syntect-server": ReferencePoint{Replicas: 1, CPU: Resource{.25, 4}, MemoryGB: Resource{2, 6}},
	},
	"docker-compose": {
		"prometheus": ReferencePoint{Replicas: 1, CPU: Resource{.5, 4}, MemoryGB: Resource{2, 8}},
		"query-runner": ReferencePoint{Replicas: 1, CPU: Resource{.5, 1}, MemoryGB: Resource{1, 1}},
		"redis-store": ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{6, 6}},
		"redis-cache": ReferencePoint{Replicas: 1, CPU: Resource{1, 1}, MemoryGB: Resource{6, 6}},
		"replacer": ReferencePoint{Replicas: 1, CPU: Resource{.5, 1}, MemoryGB: Resource{.5, .5}},
		"repo-updater": ReferencePoint{Replicas: 1, CPU: Resource{.1, 4}, MemoryGB: Resource{.5, 4}},
		"searcher": ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 2}},
		"symbols": ReferencePoint{Replicas: 1, CPU: Resource{.5, 2}, MemoryGB: Resource{.5, 4}},
		"syntect-server": ReferencePoint{Replicas: 1, CPU: Resource{.25, 4}, MemoryGB: Resource{2, 6}},
	},
}