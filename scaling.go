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
		deploymentType:    "type",
		users:             300,
		engagementRate:    100,
		repositories:      5000,
		reposize:          500,
		largeMonorepos:    5,
		largestRepoSize:   5,
		largestIndexSize:  3,
		codeintelEnabled:  "Enable",
		codeinsightEabled: "Enable",
	})
	if err != nil {
		panic(err)
	}
	// revive:disable-next-line:empty-block
	select {} // run Go forever
}

// MainView is our main component.
// revive:disable-next-line:exported
type MainView struct {
	vecty.Core
	repositories, largeMonorepos, users, engagementRate, reposize, largestRepoSize, largestIndexSize int
	deploymentType, codeintelEnabled, codeinsightEabled                                              string
}

func (p *MainView) numberInput(postLabel string, handler func(e *vecty.Event), value int, rnge scaling.Range, step int) vecty.ComponentOrHTML {
	return elem.Label(
		vecty.Markup(vecty.Style("margin-top", "10px")),
		elem.Input(
			vecty.Markup(
				vecty.Style("width", "30%"),
				event.Input(handler),
				vecty.Property("type", "number"),
				vecty.Property("value", value),
				vecty.Property("step", step),
				vecty.Property("min", rnge.Min),
				vecty.Property("max", rnge.Max),
			),
		),
		elem.Div(
			vecty.Markup(vecty.Class("post-label")),
			vecty.Text(postLabel),
		),
	)
}

func (p *MainView) rangeInput(postLabel string, handler func(e *vecty.Event), value int, rnge scaling.Range, step int) vecty.ComponentOrHTML {
	return elem.Label(
		vecty.Markup(vecty.Style("margin-top", "10px")),
		elem.Input(
			vecty.Markup(
				vecty.Style("width", "30%"),
				event.Input(handler),
				vecty.Property("type", "range"),
				vecty.Property("value", value),
				vecty.Property("step", step),
				vecty.Property("min", rnge.Min),
				vecty.Property("max", rnge.Max),
			),
		),
		elem.Div(
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
					vecty.MarkupIf(i == 0, vecty.Property("defaultChecked", "yes")),                 // pre-check the first option on every radio input
					vecty.MarkupIf(option == "kubernetes", vecty.Property("defaultChecked", "yes")), // start the estimator with kubernetes
				),
			),
			elem.Span(vecty.Text(option)),
		))
	}
	return elem.Div(
		vecty.Markup(vecty.Style("margin-top", "10px")),
		elem.Div(
			vecty.Markup(vecty.Class("radioInput"), vecty.Style("display", "inline-flex"), vecty.Style("align-items", "center")),
			elem.Strong(vecty.Markup(vecty.Style("display", "inline-flex"), vecty.Style("align-items", "center")), vecty.Text(groupName)),
			list,
		),
	)
}

func (p *MainView) inputs() vecty.ComponentOrHTML {
	return vecty.List{
		elem.Div(
			vecty.Markup(vecty.Style("padding", "20px"),
				vecty.Style("border", "1px solid")),
			p.radioInput("Deployment Type: ", []string{"docker-compose", "kubernetes"}, func(e *vecty.Event) {
				p.deploymentType = e.Value.Get("target").Get("value").String()
				vecty.Rerender(p)
			}),
			p.numberInput("users", func(e *vecty.Event) {
				p.users, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
				vecty.Rerender(p)
			}, p.users, scaling.UsersRange, 1),
			p.rangeInput(fmt.Sprint(p.engagementRate, "% engagement rate"), func(e *vecty.Event) {
				p.engagementRate, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
				vecty.Rerender(p)
			}, p.engagementRate, scaling.EngagementRateRange, 5),
			p.numberInput("repositories", func(e *vecty.Event) {
				p.repositories, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
				vecty.Rerender(p)
			}, p.repositories, scaling.RepositoriesRange, 5),
			p.numberInput("GB - the size of all repositories", func(e *vecty.Event) {
				p.reposize, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
				vecty.Rerender(p)
			}, p.reposize, scaling.TotalRepoSizeRange, 1),
			p.rangeInput(fmt.Sprint(p.largeMonorepos, " monorepos (repository larger than 1GB)"), func(e *vecty.Event) {
				p.largeMonorepos, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
				vecty.Rerender(p)
			}, p.largeMonorepos, scaling.LargeMonoreposRange, 1),
			p.numberInput("GB - the size of the largest repository", func(e *vecty.Event) {
				p.largestRepoSize, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
				vecty.Rerender(p)
			}, p.largestRepoSize, scaling.LargestRepoSizeRange, 1),
			p.radioInput("Code Insights: ", []string{"Enable", "Disable"}, func(e *vecty.Event) {
				p.codeinsightEabled = e.Value.Get("target").Get("value").String()
				vecty.Rerender(p)
			}),
			p.radioInput("Precise code intelligence: ", []string{"Enable", "Disable"}, func(e *vecty.Event) {
				p.codeintelEnabled = e.Value.Get("target").Get("value").String()
				vecty.Rerender(p)
			}),
			p.numberInput("GB - size of the largest LSIF index file", func(e *vecty.Event) {
				p.largestIndexSize, _ = strconv.Atoi(e.Value.Get("target").Get("value").String())
				vecty.Rerender(p)
			}, p.largestIndexSize, scaling.LargestIndexSizeRange, 1),
		),
	}
}

// Render implements the vecty.Component interface.
func (p *MainView) Render() vecty.ComponentOrHTML {
	estimate := (&scaling.Estimate{
		DeploymentType:   p.deploymentType,
		Repositories:     p.repositories,
		TotalRepoSize:    p.reposize,
		LargeMonorepos:   p.largeMonorepos,
		LargestRepoSize:  p.largestRepoSize,
		LargestIndexSize: p.largestIndexSize,
		Users:            p.users,
		EngagementRate:   p.engagementRate,
		CodeIntel:        p.codeintelEnabled,
		CodeInsight:      p.codeinsightEabled,
	}).Calculate().Result()

	pageExplanation := `Updating the form below will recalculate an estimate for the resources you can use to configure your Sourcegraph deployment.`

	useDefault := `Use the default values for services that are not listed below.`

	howSourcegraphScales := `[Click here to learn more about how Sourcegraph scales.](https://docs.sourcegraph.com/admin/install/kubernetes/scale)`

	explainEngagementRate := "> Engagement rate refers to the percentage of users who use Sourcegraph regularly. It is generally used for existing deployments to estimate resources."

	repoPermissionsNote := "> Repository permissions on Sourcegraph can have a noticeable impact on search performance if you have a large number of users and/or repositories on your code host.\n"
	repoPermissionsNote += ">\n"
	repoPermissionsNote += "> We suggest setting your `authorization` `ttl` values as high as you are comfortable setting it in order to reduce the chance of this (e.g. to `72h`) [in the repository permission configuration](https://docs.sourcegraph.com/admin/repo/permissions).\n"

	defaultDeployment := `> Our deployment supports instances with up to 1500 users and about 1500 repositories with one monorepo that is less than 5GB by default.`

	howToApplyRelicasResources := "> In a docker-compose deployment, edit your `docker-compose.yml` file and set `cpus` and `mem_limit` to the limits shown above.\n"
	howToApplyRelicasResources += ">\n"
	howToApplyRelicasResources += "> In Kubernetes deployments, edit the respective yaml file and update, `limits`, `requests`, and `replicas` according to the above.\n"

	dockerComposeOrK8s := "> We recommend Kubernetes for any deployments requiring > 1 service replica, but docker-compose does support service replicas and can scale up with multiple replicas as long as you can provision a sufficiently large single machine."

	return elem.Form(
		vecty.Markup(vecty.Class("estimator")),
		elem.Heading1(vecty.Text("Sourcegraph Resource Estimator")),
		&markdown{Content: []byte(pageExplanation)},
		&markdown{Content: []byte(useDefault)},
		&markdown{Content: []byte(howSourcegraphScales)},
		p.inputs(),
		&markdown{Content: estimate},
		elem.Heading3(vecty.Text("Additional information")),
		elem.Details(
			elem.Summary(vecty.Text("Default deployment size")),
			elem.Break(),
			&markdown{Content: []byte(defaultDeployment)},
		),
		elem.Details(
			elem.Summary(vecty.Text("What is engagement rate")),
			elem.Break(),
			&markdown{Content: []byte(explainEngagementRate)},
		),
		elem.Details(
			elem.Summary(vecty.Text("The recommended deployment type")),
			elem.Break(),
			&markdown{Content: []byte(dockerComposeOrK8s)},
		),
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
			elem.Summary(vecty.Text("Export result as as Markdown")),
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

// markdown is a simple component which renders the Input markdown as sanitized
// HTML into a div.
type markdown struct {
	vecty.Core
	Content []byte `vecty:"prop"`
}

// Render implements the vecty.Component interface.
func (m *markdown) Render() vecty.ComponentOrHTML {
	// Render the markdown input into HTML using Blackfriday.
	unsafeHTML := blackfriday.Run(m.Content)

	// Sanitize the HTML.
	safeHTML := string(bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML))

	// Return the HTML, which we guarantee to be safe / sanitized.
	return elem.Div(
		vecty.Markup(
			vecty.UnsafeHTML(safeHTML),
		),
	)
}
