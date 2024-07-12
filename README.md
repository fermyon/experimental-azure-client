# Overview

This repository contains the source code for the [WebAssembly component](https://github.com/orgs/fermyon/packages/container/package/wasm-pkg%2Ffermyon-experimental%2Fazure-client) we created which formats and signs HTTP calls in the way Azure requires. We have also provided Terraform that will deploy an example Storage Container and Queue instance in Azure.

### Prerequisites

We are assuming that you have familiarity with the topics below. If you are missing some familiarity with any of these topics, we have included links to helpful resources. 

- Interacting progarmmatically with Azure [Blob Storage](https://learn.microsoft.com/en-us/azure/storage/blobs/) and/or [Queue Storage](https://learn.microsoft.com/en-us/azure/storage/queues/) services
- [Writing](https://developer.fermyon.com/spin/v2/writing-apps), [building](https://developer.fermyon.com/spin/v2/build) and running a Spin application with [environment variables](https://developer.fermyon.com/spin/v2/writing-apps#adding-environment-variables-to-components)
- [Integrating a pre-built WebAssembly component into a Spin application](https://developer.fermyon.com/spin/v2/spin-application-structure)
- Making HTTP requests using [`curl`](https://curl.se/docs/tutorial.html)

### Background

As it currently stands, Spin applications written in Go, Python, and JavaScript are not able to use their respective [Azure](https://github.com/fermyon/spin/issues/2623) SDKs. As a workaround, we have built Spin components that make HTTP calls directly to Azure's API endpoints for object storage and queue services, which can be integrated into a Spin app and accessed via [internal HTTP calls](https://developer.fermyon.com/spin/v2/http-outbound#local-service-chaining).

The component was written to interact with Azure's blob and queue storage services. See Azure's [documentation](https://learn.microsoft.com/en-us/rest/api/azure/) for API information on other Azure services. 

### Benefits of migrating from serverless to Spin

While function-as-a-service products (FaaS) like Azure Functions offer remarkable scalability and simplicity, they are not without their issues. Two significant pain points for FaaS customers are cost, and cold-start times (how long it takes to have the code ready for execution). Spin solves both of these problems by offering [sub-millisecond](https://www.fermyon.com/serverless-guide/speed-and-execution-time) cold-start times, and a significant reduction in cost if deployed using [Fermyon Platform for Kubernetes](https://www.fermyon.com/platform).

### When is Spin not a good alternative to serverless?

Although Spin offers some amazing features, there are some situations for which it may not be a good fit. For example, if FaaS is not severely impacting the cost to run your applications, or if cold-start times are not meaningfully affecting the performance of your applications. In these cases, the work required to migrate existing infrastructure to Spin may not be justified by the relatively small improvements in cost and performance. Another situation where Spin may not be a good fit is if your applications rely heavily on libraries which Spin doesn't yet support. It's not impossible to find workarounds (as we have with the Azure SDK); however, there are some libraries for which we have not been able to create a workaround (see our [language guides](https://developer.fermyon.com/spin/v2/language-support-overview) for more information).

# Using the WebAssembly component

In the `spin.toml` file of the Spin application to which you want to add the Azure component, you'll need to tell Spin that you want the component to be part of your app, and you'll need to give your application permission to make HTTP calls to the Azure component:

```toml
# Don't forget that the main application needs to have permission to access the Azure client component, so don't forget to add either 'http://localhost:3000' or 'http://name-of-azure-component.spin.internal' as an allowed outbound host (see https://developer.fermyon.com/spin/v2/http-outbound#local-service-chaining for more details)

[variables]
az_account_name = { required = true, secret = true }
az_shared_key = { required = true, secret = true }

[[trigger.http]]
# For defining a custom route, see article on structuring Spin applications: https://developer.fermyon.com/spin/v2/spin-application-structure
route = "/..."
component = "name-of-azure-component"

[component.name-of-azure-component]
# Be sure to use the current version of the package. 
source = { registry = "fermyon.com", package = "fermyon-experimental:azure-client", version = " 0.1.0" }
# If the app needs to access multiple storage accounts, use "https://*.{{blob|queue}}.core.windows.net"
allowed_outbound_hosts = [
    "https://{{ az_account_name }}.blob.core.windows.net", 
    "https://{{ az_account_name }}.queue.core.windows.net",
]

[component.name-of-azure-component.variables]
az_account_name = "{{ az_account_name }}"
az_shared_key = "{{ az_shared_key }}"
```

Once these entries have been added to the `spin.toml` file, you can run `spin build`.

# Building from source

### Requirements

- Latest version of [Spin](https://developer.fermyon.com/spin/v2/install)
- Latest version of [Go](https://go.dev/doc/install)
- Latest version of [TinyGo](https://tinygo.org/getting-started/install/)


### Building the component:

Navigate to the directory containing the codefiles, then run the below commands:

```bash
# Installing dependencies
go mod download
# Building the component
spin build
```

# Running the application

### Export environment variables

In your terminal, export the below variables:

```bash
export SPIN_VARIABLE_AZ_ACCOUNT_NAME=YOUR_ACCOUNT_NAME
export SPIN_VARIABLE_AZ_SHARED_KEY=YOUR_SHARED_KEY
```

Notice that the environment variables are formatted `SPIN_VARIABLE_UPPERCASE_VARIABLE_NAME`. This is the format required by Spin to read environment variables properly. As can be seen in the `spin.toml` file, the Spin application accesses the variables as `lowercase_variable_name`. 

Once the environment variables have been exported, you can run `spin up`.

# Interacting with the application:

The curl request examples below are for standalone Azure components. If trying to interact with the Azure component from within Spin, the commands will look a little different:

```golang
// Place blob
method := "PUT"
endpoint := "http://name-of-azure-component.spin.internal/container-name/path/to/your/blob"
bodyData := []byte("Hello, Azure!")

req, err := http.NewRequest(method, endpoint, bytes.NewReader(bodyData))
if err != nil {
    panic(err)
}

req.Header.Set("x-az-service", "blob")

resp, err := spinhttp.Send(req)
```

## List blobs:

```bash
curl \
    -H 'x-az-service: blob' \
    "http://127.0.0.1:3000/container-name?restype=container&comp=list"
```

## Get blob:

```bash
curl \
    -o file_name.extension \
    -H 'x-az-service: blob' \
    http://127.0.0.1:3000/container-name/path/to/your/blob
```

## Delete blob:

```bash
curl \
    --request DELETE \
    -H 'x-az-service: blob' \
    http://127.0.0.1:3000/container-name/path/to/your/blob
```

## Place blob:

```bash
curl \
    --request PUT \
    -H 'x-az-service: blob' \
    --data-binary @/path/to/file \
    http://127.0.0.1:3000/container-name/path/to/your/blob
```

## List queues:

```bash
curl \
    -H 'x-az-service: queue' \
    "http://127.0.0.1:3000?comp=list"
```
## Get queue messages:

```bash
curl \
    -H 'x-az-service: queue' \
    http://127.0.0.1:3000/your-queue-name/messages
```

## Delete queue message:

```bash
# The message-id and pop-receipt string values can be retrieved via getting messages from the queue.
curl \
    --request DELETE \
    -H 'x-az-service: queue' \
    "http://127.0.0.1:3000/your-queue-name/messages/your-message-id?popreceipt=your-pop-receipt-value"
```

## Place queue message: 

```bash
# Per their documentation, the request body needs to be formatted using the XML as follows:
# <QueueMessage>
#   <MessageText>YourMessageHere</MessageText>
# </QueueMessage>
curl \
    --request POST \
    -H 'x-az-service: queue' \
    --data-binary @path/to/your/xml/message \
    http://127.0.0.1:3000/your-queue-name/messages
```