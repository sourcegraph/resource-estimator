package main

import (
	"fmt"
	"strconv"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
	"github.com/sourcegraph/resource-estimator/internal/scaling"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

func main() {
	vecty.SetTitle("Resource estimator - Sourcegraph")
	err := vecty.RenderInto("#root", &MainView{
		deploymentType: "estimated",
		repositories:   300,
		largeMonorepos: 0,
		users:          100,
		engagementRate: 50,
	})
	if err != nil {
		panic(err)
	}
	select {} // run Go forever
}

// MainView is our main component.
type MainView struct {
	vecty.Core
	repositories, largeMonorepos, users, engagementRate int
	deploymentType                                      string
}

func (p *MainView) numberInput(postLabel string, handler func(e *vecty.Event), value int, rnge scaling.Range, step int) vecty.ComponentOrHTML {
	return elem.Label(
		elem.Input(
			vecty.Markup(
				event.Input(handler),
				vecty.Property("type", "number"),
				vecty.Property("value", value),
				vecty.Property("step", step),
				vecty.Property("min", rnge.Min),
				vecty.Property("max", rnge.Max),
			),
		),
		elem.Span(
			vecty.Markup(vecty.Class("post-label")),
			vecty.Text(postLabel),
		),
	)
}

func (p *MainView) rangeInput(postLabel string, handler func(e *vecty.Event), value int, rnge scaling.Range, step int) vecty.ComponentOrHTML {
	return elem.Label(
		elem.Input(
			vecty.Markup(
				event.Input(handler),
				vecty.Property("type", "range"),
				vecty.Property("value", value),
				vecty.Property("step", step),
				vecty.Property("min", rnge.Min),
				vecty.Property("max", rnge.Max),
			),
		),
		elem.Span(
			vecty.Markup(vecty.Class("post-label")),
			vecty.Text(postLabel),
		),
	)
}

func (p *MainView) radioInput(groupName string, options []string, handler func(e *vecty.Event)) vecty.ComponentOrHTML {
	var list vecty.List
	for i, option := range options {
		list = append(list, elem.Label(
			elem.Input(
				vecty.Markup(
					event.Input(handler),
					vecty.Property("type", "radio"),
					vecty.Property("value", option),
					vecty.Property("name", groupName),
					vecty.MarkupIf(i == 0, vecty.Property("defaultChecked", "yes")),
				),
			),
			elem.Span(vecty.Text(option)),
		))
	}
	return elem.Div(
		vecty.Markup(vecty.Class("radioInput")),
		elem.Strong(vecty.Text(groupName)),
		list,
	)
}

func (p *MainView) inputs() vecty.ComponentOrHTML {
	return vecty.List{
		elem.Heading3(vecty.Text("Inputs")),
		p.numberInput("repositories", func(e *vecty.Event) {
			p.repositories, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
			vecty.Rerender(p)
		}, p.repositories, scaling.RepositoriesRange, 5),
		p.numberInput("users", func(e *vecty.Event) {
			p.users, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
			vecty.Rerender(p)
		}, p.users, scaling.UsersRange, 1),
		p.rangeInput(fmt.Sprint(p.largeMonorepos, " large monorepos"), func(e *vecty.Event) {
			p.largeMonorepos, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
			vecty.Rerender(p)
		}, p.largeMonorepos, scaling.LargeMonoreposRange, 1),
		p.rangeInput(fmt.Sprint(p.engagementRate, "% engagement rate"), func(e *vecty.Event) {
			p.engagementRate, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
			vecty.Rerender(p)
		}, p.engagementRate, scaling.EngagementRateRange, 5),
		p.radioInput("Deployment Type: ", []string{"estimated", "docker-compose", "kubernetes"}, func(e *vecty.Event) {
			p.deploymentType = e.Value.Get("target").Get("value").String()
			vecty.Rerender(p)
		}),
	}
}

// Render implements the vecty.Component interface.
func (p *MainView) Render() vecty.ComponentOrHTML {
	estimate := (&scaling.Estimate{
		DeploymentType: p.deploymentType,
		Repositories:   p.repositories,
		LargeMonorepos: p.largeMonorepos,
		Users:          p.users,
		EngagementRate: p.engagementRate,
	}).Calculate().Markdown()

	repoPermissionsNote := "> Repository permissions on Sourcegraph can have a noticeable impact on search performance if you have a large number of users and/or repositories on your code host.\n"
	repoPermissionsNote += ">\n"
	repoPermissionsNote += "> We suggest setting your `authorization` `ttl` values as high as you are comfortable setting it in order to reduce the chance of this (e.g. to `72h`) [in the repository permission configuration](https://docs.sourcegraph.com/admin/repo/permissions).\n"

	pageExplanation := `Enter your inputs below and the page will calculate an estimate for what deployment you should start out with, then later [learn more about how Sourcegraph scales](https://docs.sourcegraph.com/admin/install/kubernetes/scale).`

	howToApplyRelicasResources := "> In a docker-compose deployment, edit your `docker-compose.yml` file and set `cpus` and `mem_limit` to the limits shown above.\n"
	howToApplyRelicasResources += ">\n"
	howToApplyRelicasResources += "> In Kubernetes deployments, edit the respective yaml file and update, `limits`, `requests`, and `replicas` according to the above.\n"

	return elem.Form(
		vecty.Markup(vecty.Class("estimator")),
		elem.Heading1(vecty.Text("Sourcegraph resource estimator")),
		&markdown{Content: []byte(pageExplanation)},
		p.inputs(),
		&markdown{Content: estimate},
		elem.Heading3(vecty.Text("Additional information")),
		elem.Details(
			elem.Summary(vecty.Text("How to apply these changes to your deployment")),
			elem.Break(),
			&markdown{Content: []byte(howToApplyRelicasResources)},
		),
		elem.Details(
			elem.Summary(vecty.Text("If you plan to enforce repository permissions on Sourcegraph")),
			elem.Break(),
			&markdown{Content: []byte(repoPermissionsNote)},
		),
		elem.Details(
			elem.Summary(vecty.Text("Copy this estimate as Markdown")),
			elem.Break(),
			elem.TextArea(
				vecty.Markup(vecty.Class("copy-as-markdown")),
				vecty.Text(string(estimate)),
			),
		),
		elem.Break(),
		elem.Paragraph(
			elem.Strong(vecty.Text("Questions or concerns? ")),
			elem.Anchor(
				vecty.Markup(prop.Href("mailto:support@sourcegraph.com")),
				vecty.Text("Get help from an engineer"),
			),
		),
	)
}

const repositoryPermissionsNote = `Repository permissions on Sourcegraph can have a noticeable impact on search performance if you have a large number of users and/or repositories on your code host.`

// markdown is a simple component which renders the Input markdown as sanitized
// HTML into a div.
type markdown struct {
	vecty.Core
	Content []byte `vecty:"prop"`
}

// Render implements the vecty.Component interface.
func (m *markdown) Render() vecty.ComponentOrHTML {
	// Render the markdown input into HTML using Blackfriday.
	unsafeHTML := blackfriday.Run([]byte(m.Content))

	// Sanitize the HTML.
	safeHTML := string(bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML))

	// Return the HTML, which we guarantee to be safe / sanitized.
	return elem.Div(
		vecty.Markup(
			vecty.UnsafeHTML(safeHTML),
		),
	)
}
