// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"text/template"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
)

// Define the IstioOperator template which is used to insert the generated YAML values.
//
// NOTE: The go template rendering doesn't properly indent the multi-line YAML value
// For example, the template fragment only indents the fist line of values
//    global:
//      {{.Values}}
// so the result is
//    global:
//      line1:
// line2:
//   line3:
// etc...
//
// A solution is to pre-indent each line of the values then insert it at column 0 as follows:
//    global:
// {{.Values}}
// See the leftMargin usage in the code
//
const istioCrTempate = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
      - name: istio-egressgateway
        enabled: true

  # Global values passed through to helm global.yaml.
  # Please keep this in sync with manifests/charts/global.yaml
  values:
    global:
{{.Values}}
`

// templateValues needed for template rendering
type templateValues struct {
	Values string
}

// BuildIstioOperatorYaml builds the IstioOperator CR YAML that will be passed as an override to istioctl
// Transform the Verrazzano CR IstioComponent provided by the user onto an IstioOperator formatted YAML
func BuildIstioOperatorYaml(comp *vzapi.IstioComponent) (string, error) {
	//
	const leftMargin = 6

	// Build a list of YAML strings from the IstioComponent initargs, one for each arg.
	var yamls []string
	for _, arg := range comp.IstioInstallArgs {
		values := arg.ValueList
		if len(values) == 0 {
			values = []string{arg.Value}
		}
		yaml, err := vzyaml.Expand(leftMargin, arg.Name, values...)
		if err != nil {
			return "", err
		}
		yamls = append(yamls, yaml)
	}

	// Merge the YAML strings, second has precedence over first, third over second, and so forth.
	merged, err := vzyaml.ReplacementMerge(yamls...)
	if err != nil {
		return "", err
	}

	// Combine the merged YAML with the template to provide the IstioOperator YAML
	// First create the template then render it.
	t, err := template.New("image").Parse(istioCrTempate)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	tInput := templateValues{Values: merged}
	err = t.Execute(&rendered, tInput)
	if err != nil {
		return "", err
	}

	return rendered.String(), nil
}
