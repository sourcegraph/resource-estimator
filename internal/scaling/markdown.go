package scaling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
)

func (e *Estimate) MarkdownExport() []byte {
	var buf bytes.Buffer
	// Summary of the output
	fmt.Fprintf(&buf, "### Estimate summary\n")
	fmt.Fprintf(&buf, "\n")
	if e.ContactSupport {
		fmt.Fprintf(&buf, "**Estimation is currently not available for your instance size. Please [contact support](mailto:support@sourcegraph.com) for further assists.**\n")
	} else {
		fmt.Fprintf(&buf, "* **Instance Size:** %v\n", e.InstanceSize)
		fmt.Fprintf(&buf, "* **Estimated vCPUs:** %v\n", e.TotalCPU)
		fmt.Fprintf(&buf, "* **Estimated Memory:** %vg\n", e.TotalMemoryGB)
		fmt.Fprintf(&buf, "* **Estimated Minimum Volume Size:** %vg\n", e.TotalStorageSize)
		fmt.Fprintf(&buf, "* **Recommend Deployment Type:** [%v](https://docs.sourcegraph.com/admin/deploy#deployment-types)\n", e.RecommendedDeploymentType)

		fmt.Fprintf(&buf, "\n<small>**Note:** The estimated values include default values for services that are not listed in the estimator, like otel-collector and repo-updater for example. The default values for the non-displaying services should work well with instances of all sizes.</small>\n")
		if e.EngagedUsers < 650/2 && e.AverageRepositories < 1500/2 {
			//nolint:staticcheck
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

		fmt.Fprintf(&buf, "| Service | Replica | CPU requests | CPU limits | MEM requests | MEM limits  | Storage |\n")
		fmt.Fprintf(&buf, "|-------|:-------:|:-------:|:-------:|:-------:|:-------:|:-------:|\n")

		var names []string
		for service := range e.Services {
			names = append(names, service)
		}
		sort.Strings(names)

		for _, service := range names {
			ref := e.Services[service]
			def := defaults[service][e.DeploymentType]
			plus := ""
			serviceName := fmt.Sprint("**", ref.Label, "**", "</br><small>(pod: ", ref.PodName, ")</small>")
			replicas := "n/a"
			cpuRequest := "n/a"
			cpuLimit := "n/a"
			memoryGBRequest := "n/a"
			memoryGBLimit := "n/a"
			//nolint:ineffassign
			ephRequest := "-"
			//nolint:ineffassign
			ephLimit := "-"
			pvc := "-"
			if !ref.ContactSupport {
				if e.DeploymentType == "docker-compose" {
					serviceName = "**" + ref.NameInDocker + "**"
					replicas = fmt.Sprint("1", plus)
					cpuRequest = "-"
					cpuLimit = fmt.Sprint(ref.Resources.Limits.CPU*float64(ref.Replicas), plus)
					memoryGBRequest = "-"
					memoryGBLimit = fmt.Sprint(ref.Resources.Limits.MEM*float64(ref.Replicas), "g", plus)
					//nolint:ineffassign
					ephRequest = "-"
					//nolint:ineffassign
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
						ephRequest = fmt.Sprint(ref.Resources.Requests.EPHS)
						ephLimit = fmt.Sprint(ref.Resources.Limits.EPHS, plus, "ꜝ")
						pvc = fmt.Sprint(ephRequest, "/", ephLimit)
					}
					// }
					if ref.Resources.Limits.CPU != def.Resources.Limits.CPU {
						cpuLimit += "ꜝ"
					}
					if ref.Resources.Limits.MEM != def.Resources.Limits.MEM {
						memoryGBLimit += "ꜝ"
					}
				}
			}
			fmt.Fprintf(
				&buf,
				"| %v | %v | %v | %v | %v | %v | %v |\n",
				serviceName,
				replicas,
				cpuRequest,
				cpuLimit,
				memoryGBRequest,
				memoryGBLimit,
				pvc,
			)
		}
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "> ꜝ<small> This is a non-default value.</small>\n")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "\n")
	}
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
	s := strings.ReplaceAll(string(y), `"`, "'")
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
	s := strings.ReplaceAll(string(y), `"`, "")
	return s
}
