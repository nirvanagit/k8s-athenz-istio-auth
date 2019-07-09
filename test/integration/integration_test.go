package integration

import (
	"crypto/tls"
	"log"
	"net"
	"testing"
	"time"

	openapi "github.com/go-openapi/spec"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubeopenapi "k8s.io/kube-openapi/pkg/common"
)

func TestMain(m *testing.M) {
	log.Println("in main")
	restConfig := &rest.Config{}
	restConfig.Host = "127.0.0.0:9999"

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Println("in here")
		log.Panicln(err.Error())
		return
	}

	Scheme := runtime.NewScheme()
	config := server.NewConfig(serializer.NewCodecFactory(Scheme))
	config.Authorization.Authorizer = authorizerfactory.NewAlwaysAllowAuthorizer()
	listener, err := net.Listen("tcp4", "")
	if err != nil {
		log.Panicln(err)
	}
	cert, err := tls.LoadX509KeyPair("/Users/mcieplak/.athenz/cert", "/Users/mcieplak/.athenz/key")
	if err != nil {
		log.Panicln(err)
	}
	config.SecureServing = &server.SecureServingInfo{
		Listener: listener,
		Cert:     &cert,
	}
	config.LoopbackClientConfig = restConfig
	config.OpenAPIConfig = server.DefaultOpenAPIConfig(testGetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(runtime.NewScheme()))

	stopCh := make(chan struct{})
	shared := informers.NewSharedInformerFactory(clientset, 0)
	apiServer, err := config.Complete(shared).New("api-server", server.NewEmptyDelegate())
	if err != nil {
		log.Println("error")
		log.Panicln(err.Error())
		return
	}
	log.Println("before run")
	apiServer.PrepareRun().Run(stopCh)
	log.Println("after run")

	//config := server.RecommendedConfig{}
	//config.Serializer = &fakeNegotiatedSerializer{}
	//config.LoopbackClientConfig = restConfig
	//config.OpenAPIConfig = nil
	//config.SecureServing = nil
	//foo := config.Complete()

	//foo := server.CompletedConfig{}
	//delegate := server.NewEmptyDelegate()
	//if delegate == nil {
	//	log.Println("delegate is nil")
	//	return
	//}
	//
	//log.Println("here")
	//_, err := foo.New("api-server", delegate)
	//log.Println("after here")
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	//
	//stopCh := make(chan struct{})
	//err = srv.PrepareRun().Run(stopCh)
	//if err != nil {
	//	log.Println(err)
	//	return
	//}

	m.Run()
	time.Sleep(time.Second)
	log.Println("done")
}

func testGetOpenAPIDefinitions(_ kubeopenapi.ReferenceCallback) map[string]kubeopenapi.OpenAPIDefinition {
	return map[string]kubeopenapi.OpenAPIDefinition{
		"k8s.io/apimachinery/pkg/apis/meta/v1.Status":          {},
		"k8s.io/apimachinery/pkg/apis/meta/v1.APIVersions":     {},
		"k8s.io/apimachinery/pkg/apis/meta/v1.APIGroupList":    {},
		"k8s.io/apimachinery/pkg/apis/meta/v1.APIGroup":        buildTestOpenAPIDefinition(),
		"k8s.io/apimachinery/pkg/apis/meta/v1.APIResourceList": {},
	}
}

func buildTestOpenAPIDefinition() kubeopenapi.OpenAPIDefinition {
	return kubeopenapi.OpenAPIDefinition{
		Schema: openapi.Schema{
			SchemaProps: openapi.SchemaProps{
				Description: "Description",
				Properties:  map[string]openapi.Schema{},
			},
			VendorExtensible: openapi.VendorExtensible{
				Extensions: openapi.Extensions{
					"x-kubernetes-group-version-kind": []map[string]string{
						{
							"group":   "",
							"version": "v1",
							"kind":    "Getter",
						},
						{
							"group":   "batch",
							"version": "v1",
							"kind":    "Getter",
						},
						{
							"group":   "extensions",
							"version": "v1",
							"kind":    "Getter",
						},
					},
				},
			},
		},
	}
}
