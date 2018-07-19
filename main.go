package main

import (
	"fmt"
	"log"

	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	hapi "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/storage/driver"
	"k8s.io/helm/pkg/tiller/environment"
	"k8s.io/helm/pkg/timeconv"
)

func main() {
	clientset, err := kube.New(nil).ClientSet()
	if err != nil {
		panic(err)
	}

	// create a release
	log.Print("Constructing Helm Release...")
	release := BuildRelease("core-paas-namespaces")

	log.Print("\n")
	log.Print("Installing Helm Chart...")
	cfgmaps := driver.NewConfigMaps(clientset.Core().ConfigMaps(environment.DefaultTillerNamespace))
	cfgmaps.Log = helmPrinter
	releases := storage.Init(cfgmaps)
	releases.Log = helmPrinter
	if err := releases.Create(release); err != nil {
		panic(err)
	}
}

func helmPrinter(s string, params ...interface{}) {
	params = append([]interface{}{s}, params...)
	log.Print(params...)
}

func BuildRelease(name string) *hapi.Release {
	templates, manifest := buildTemplates()
	return &hapi.Release{
		Name:      name,
		Info:      buildReleaseInfo(),
		Chart:     buildReleaseChart(templates),
		Manifest:  manifest,
		Version:   1,
		Namespace: "core-paas",
	}
}

func buildTemplates() ([]*chart.Template, string) {
	manifest := ""
	var namespaces []string
	namespaces = append(namespaces, "api-tooling")
	// kdev, kqa only
	//namespaces = append(namespaces, "cs")
	namespaces = append(namespaces, "design-center")
	namespaces = append(namespaces, "exchange")
	// kqa only
	//namespaces = append(namespaces, "poseidon")

	// namespace (7) + secret (5) + resourcequotas (6)
	var templates []*chart.Template

	// iterate over namespace templates
	for _, name := range namespaces {
		manifest += addNamespaceChartTemplate(templates, name)
	}

	// Add in "core-paas" (current namespace) for secret and resource quota
	namespaces = append(namespaces, "core-paas")

	// iterate over secret templates
	for _, name := range namespaces {
		manifest += addSecretChartTemplate(templates, name)
	}

	// add in default for resource quota only
	// namespaces = append(namespaces, "default")
	//
	// // iterate over resourcequota templates
	// for _, name := range namespaces {
	// 	manifest += addResourceQuotaChartTemplate(templates, name)
	// }

	return templates, manifest
}

func addNamespaceChartTemplate(templates []*chart.Template, name string) string {
	template := fmt.Sprintf("apiVersion: v1\n"+
		"kind: Namespace\n"+
		"metadata:\n"+
		"  name: \"%v\"", name)

	filename := "core-paas-namespaces/templates/namespaces.yaml"
	return generateTemplate(templates, filename, template)
}

func addSecretChartTemplate(templates []*chart.Template, name string) string {
	template := fmt.Sprintf("apiVersion: v1\n"+
		"kind: Secret\n"+
		"type: kubernetes.io/dockercfg\n"+
		"\n"+
		"metadata:\n"+
		"  name: \"devdocker-registrykey\"\n"+
		"  namespace: \"%v\"\n"+
		"\n"+
		"data:\n"+
		"  .dockercfg: eyJkZXZkb2NrZXIubXVsZXNvZnQuY29tOjE4MDc4Ijp7InVzZXJuYW1lIjoiYXJtLXB1Ymxpc2hlciIsInBhc3N3b3JkIjoiQWV3Nndlczl0aGFoNHdhIiwiZW1haWwiOiJ2YWxreXJAbXVsZXNvZnQuY29tIiwiYXV0aCI6IllYSnRMWEIxWW14cGMyaGxjanBCWlhjMmQyVnpPWFJvWVdnMGQyRT0ifX0=", name)

	filename := "core-paas-namespaces/templates/docker-registry-secret.yaml"
	return generateTemplate(templates, filename, template)
}

func addResourceQuotaChartTemplate(templates []*chart.Template, name string) string {
	template := fmt.Sprintf("apiVersion: v1\n"+
		"kind: ResourceQuota\n"+
		"metadata:\n"+
		"  name: compute-resources\n"+
		"  namespace: \"%v\"\n"+
		"spec:\n"+
		"  hard:\n"+
		"    requests.cpu: 2\n"+
		"    requests.memory: 8Gi\n"+
		"    limits.cpu: 5\n"+
		"    limits.memory: 10Gi", name)

	filename := "core-paas-namespaces/templates/resource-quotas.yaml"
	return generateTemplate(templates, filename, template)
}

func generateTemplate(templates []*chart.Template, filename string, template string) string {
	templates = append(templates, &chart.Template{
		Name: filename,
		Data: []byte(template),
	})
	return fmt.Sprintf("\n---\n# Source: %v\n%v", filename, template)
}

func buildReleaseInfo() *hapi.Info {
	info := &hapi.Info{
		Status: &hapi.Status{
			Code: hapi.Status_DEPLOYED,
		},
		FirstDeployed: timeconv.Now(),
		LastDeployed:  timeconv.Now(),
		Description:   "Transferred ownership to Helm via helm-take-ownership",
	}

	return info
}

func buildReleaseChart(templates []*chart.Template) *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        "core-paas-namespaces",
			Version:     "0.4.0-pre.1",
			Description: "Chart built by helm-take-ownership",
			ApiVersion:  "v1",
		},
		Templates: templates,
	}
}
