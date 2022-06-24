package scaling_test

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/hexops/autogold"
	"github.com/sourcegraph/resource-estimator/internal/scaling"
)

func TestEstimate(t *testing.T) {
	cases := []struct {
		Name string
		scaling.Estimate
	}{{
		Name: "default",
		Estimate: scaling.Estimate{
			DeploymentType: "estimated",
			Repositories:   300,
			LargeMonorepos: 0,
			Users:          100,
			EngagementRate: 50,
		},
	}, {
		Name: "monorepo",
		Estimate: scaling.Estimate{
			DeploymentType: "estimated",
			Repositories:   0,
			LargeMonorepos: 1,
			Users:          100,
			EngagementRate: 50,
		},
	}}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got := string(tc.Estimate.Calculate().Markdown())
			autogold.Equal(t, got)
		})
	}
}

// This test will ensure that the outputs of calculate don't break any known
// invariants we expect. We do a mix of random inputs and some exhaustive
// checks.
func TestInvariants(t *testing.T) {
	f := func(e *scaling.Estimate) bool {
		e = e.Calculate()

		if e.Services["zoekt-webserver"].Replicas != e.Services["zoekt-indexserver"].Replicas {
			t.Log("zoekt-webserver replicas != zoekt-indexserver replicas but they live in the same pod")
			t.Log(string(e.Markdown()))
			return false
		}

		return true
	}

	for repos := int(scaling.RepositoriesRange.Min); repos <= int(scaling.RepositoriesRange.Max); repos++ {
		if !f(&scaling.Estimate{Repositories: repos, Users: 100, EngagementRate: 50}) {
			t.Fatal()
		}
	}

	config := &quick.Config{
		Values: func(args []reflect.Value, r *rand.Rand) {
			e := &scaling.Estimate{
				Repositories:   randRange(r, scaling.RepositoriesRange),
				LargeMonorepos: randRange(r, scaling.LargeMonoreposRange),
				Users:          randRange(r, scaling.UsersRange),
				EngagementRate: randRange(r, scaling.EngagementRateRange),
			}
			args[0] = reflect.ValueOf(e)
		},
	}

	if err := quick.Check(f, config); err != nil {
		t.Fatal(err)
	}
}

func randRange(r *rand.Rand, v scaling.Range) int {
	min := int32(v.Min)
	max := int32(v.Max)
	return int(min + r.Int31n(max-min+1))
}