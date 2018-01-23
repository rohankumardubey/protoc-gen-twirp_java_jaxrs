package main

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

type generator struct {
	Request  *plugin.CodeGeneratorRequest
	Response *plugin.CodeGeneratorResponse

	output *bytes.Buffer
	indent string
}

func newGenerator(req *plugin.CodeGeneratorRequest) *generator {
	return &generator{
		Request:  req,
		Response: nil,
		output:   bytes.NewBuffer(nil),
		indent:   "",
	}
}

func (g *generator) Generate() error {
	g.Response = &plugin.CodeGeneratorResponse{}

	for _, file := range g.getProtoFiles() {
		g.processFile(file)
	}

	return nil
}

func (g *generator) processFile(file *descriptor.FileDescriptorProto) {
	for _, service := range file.GetService() {
		out := g.generateServiceInterface(file, service)
		g.Response.File = append(g.Response.File, out)

		out = g.generateServiceClient(file, service)
		g.Response.File = append(g.Response.File, out)
	}
}

func (g *generator) generateServiceClient(file *descriptor.FileDescriptorProto, service *descriptor.ServiceDescriptorProto) *plugin.CodeGeneratorResponse_File {
	pkg := getJavaPackage(file)

	g.P(`// Code generated by protoc-gen-twirp_java_jaxrs, DO NOT EDIT.`)
	g.P(`// source: `, file.GetName())
	g.P()
	g.P(`package `, pkg, `;`)
	g.P()
	g.P(`import java.io.InputStream;`)
	g.P(`import java.util.function.Function;`)
	g.P(`import javax.ws.rs.client.Entity;`)
	g.P(`import javax.ws.rs.client.WebTarget;`)
	g.P(`import javax.ws.rs.core.Response;`)
	g.P(`import javax.ws.rs.core.StreamingOutput;`)
	g.P(`import com.google.protobuf.MessageLite;`)
	g.P()

	// TODO add comment

	serviceClass := getJavaServiceClientClassName(file, service)
	servicePath := g.getServicePath(file, service)
	interfaceClass := fmt.Sprintf("%s.%s", pkg, getJavaServiceClassName(file, service))

	g.P(`public class `, serviceClass, ` implements `, interfaceClass, ` {`)
	g.P(`  private static final String PATH = "/twirp/`, servicePath, `";`)
	g.P(`  private final WebTarget target;`)
	g.P()
	g.P(`  public `, serviceClass, `(WebTarget target) {`)
	g.P(`    this.target = target;`)
	g.P(`  }`)
	g.P()
	g.P(`  private <T> T _parseSafely(InputStream input, FunctionE<InputStream, T> fn) {`)
	g.P(`    try {`)
	g.P(`      return fn.apply(input);`)
	g.P(`    } catch (Exception e) {`)
	g.P(`      throw new RuntimeException(e);`)
	g.P(`    }`)
	g.P(`  }`)
	g.P()
	g.P(`  @FunctionalInterface`)
	g.P(`  interface FunctionE<A, B> {`)
	g.P(`    B apply(A input) throws Exception;`)
	g.P(`  }`)
	g.P()
	g.P(`  private <R> R _call(String path, MessageLite request, Function<InputStream, R> parser) {`)
	g.P(`    Response response = target.path(path)`)
	g.P(`        .request("application/protobuf")`)
	g.P(`        .post(Entity.entity((StreamingOutput) request::writeTo, "application/protobuf"));`)
	g.P(`    InputStream body = response.readEntity(InputStream.class);`)
	g.P(`    return parser.apply(body);`)
	g.P(`  }`)

	for _, method := range service.GetMethod() {
		inputType := getJavaType(file, method.GetInputType())
		outputType := getJavaType(file, method.GetOutputType())
		methodName := lowerCamelCase(method.GetName())
		methodPath := camelCase(method.GetName())

		g.P()
		// add comment
		g.P(`  @Override`)
		g.P(`  public `, outputType, ` `, methodName, `(`, inputType, ` request) {`)
		g.P(`    Function<InputStream, `, outputType, `> parser =`)
		g.P(`        (input) -> _parseSafely(input, `, outputType, `::parseFrom);`)
		g.P(`    return _call(PATH + "/`, methodPath, `", request, parser);`)
		g.P(`  }`)
	}

	g.P(`}`)

	out := &plugin.CodeGeneratorResponse_File{}
	out.Name = proto.String(getJavaServiceClientClassFile(file, service))
	out.Content = proto.String(g.output.String())
	g.Reset()

	return out
}

func (g *generator) generateServiceInterface(file *descriptor.FileDescriptorProto, service *descriptor.ServiceDescriptorProto) *plugin.CodeGeneratorResponse_File {
	pkg := getJavaPackage(file)

	g.P(`// Code generated by protoc-gen-twirp_java_jaxrs, DO NOT EDIT.`)
	g.P(`// source: `, file.GetName())
	g.P()
	g.P(`package `, pkg, `;`)
	g.P()

	// TODO add comment

	serviceClass := getJavaServiceClassName(file, service)

	g.P(`public interface `, serviceClass, ` {`)

	for _, method := range service.GetMethod() {
		inputType := getJavaType(file, method.GetInputType())
		outputType := getJavaType(file, method.GetOutputType())
		methodName := lowerCamelCase(method.GetName())

		// add comment
		g.P(`  `, outputType, ` `, methodName, `(`, inputType, ` request);`)
	}

	g.P(`}`)

	out := &plugin.CodeGeneratorResponse_File{}
	out.Name = proto.String(getJavaServiceClassFile(file, service))
	out.Content = proto.String(g.output.String())
	g.Reset()

	return out
}

func (g *generator) Reset() {
	g.indent = ""
	g.output.Reset()
}

func (g *generator) In() {
	g.indent += "  "
}

func (g *generator) Out() {
	g.indent = g.indent[2:]
}

func (g *generator) P(str ...string) {
	for _, v := range str {
		g.output.WriteString(v)
	}
	g.output.WriteByte('\n')
}

func (g *generator) getProtoFiles() []*descriptor.FileDescriptorProto {
	files := make([]*descriptor.FileDescriptorProto, 0)
	for _, fname := range g.Request.GetFileToGenerate() {
		for _, proto := range g.Request.GetProtoFile() {
			if proto.GetName() == fname {
				files = append(files, proto)
			}
		}
	}
	return files
}

func (g *generator) getServicePath(file *descriptor.FileDescriptorProto, service *descriptor.ServiceDescriptorProto) string {
	name := camelCase(service.GetName())
	pkg := file.GetPackage()
	if pkg != "" {
		name = pkg + "." + name
	}
	return name
}