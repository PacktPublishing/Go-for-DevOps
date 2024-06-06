


# Go for DevOps

<a href="https://www.packtpub.com/product/go-for-devops/9781801818896?utm_source=github&utm_medium=repository&utm_campaign=9781801818896"><img src="https://static.packt-cdn.com/products/9781801818896/cover/smaller" alt="Go for DevOps" height="256px" align="right"></a>

This is the code repository for [Go for DevOps](https://www.packtpub.com/product/go-for-devops/9781801818896?utm_source=github&utm_medium=repository&utm_campaign=9781801818896), published by Packt.

**Learn how to use the Go language to automate servers, the cloud, Kubernetes, GitHub, Packer, and Terraform**

## What is this book about?
Go is the go-to language for DevOps libraries and services, and without it, achieving fast and safe automation is a challenge. With the help of Go for DevOps, you'll learn how to deliver services with ease and safety, becoming a better DevOps engineer in the process. 

This book covers the following exciting features:
* Understand the basic structure of the Go language to begin your DevOps journey
* Interact with filesystems to read or stream data
* Communicate with remote services via REST and gRPC
* Explore writing tools that can be used in the DevOps environment
* Develop command-line operational software in Go
* Work with popular frameworks to deploy production software
* Create GitHub actions that streamline your CI/CD process
* Write a ChatOps application with Slack to simplify production visibility

If you feel this book is for you, get your [copy](https://www.amazon.com/dp/1801818894) today!

<a href="https://www.packtpub.com/?utm_source=github&utm_medium=banner&utm_campaign=GitHubBanner"><img src="https://raw.githubusercontent.com/PacktPublishing/GitHub/master/GitHub.png" 
alt="https://www.packtpub.com/" border="5" /></a>

## Instructions and Navigations
All of the code is organized into folders. For example, Chapter02.

The code will look like the following:
```
packer {
 required_plugins {
 amazon = {
 version = ">= 0.0.1"

```

**Following is what you need for this book:**
This book is for Ops and DevOps engineers who would like to use Go to develop their own DevOps tooling or integrate custom features with DevOps tools such as Kubernetes, GitHub Actions, HashiCorp Packer, and Terraform. Experience with some type of programming language, but not necessarily Go, is necessary to get started with this book.

With the following software and hardware list you can run all code files present in the book (Chapter 1-16).
### Software and Hardware List
| Chapter  | Software required | OS required |
| -------- | ------------------------------------ | ----------------------------------- |
| 1-16     | Go 1.18           | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Packer            | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Terraform         | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Kubernetes        | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Docker            | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Tilt              | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Protocol Buffers  | Windows, Mac OS X, and Linux (Any) |
| 1-16     | gPRC,ctlptl       | Windows, Mac OS X, and Linux (Any) |
| 1-16     | But CLI           | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Operator SDK      | Windows, Mac OS X, and Linux (Any) |
| 1-16     | Azure CLI, KinD   | Windows, Mac OS X, and Linux (Any) |

We also provide a PDF file that has color images of the screenshots/diagrams used in this book. [Click here to download it](https://static.packt-cdn.com/downloads/9781801818896_ColorImages.pdf).

### Related products
* Learning DevOps - Second Edition [[Packt]](https://www.packtpub.com/product/learning-devops-second-edition/9781801818964?utm_source=github&utm_medium=repository&utm_campaign=9781801818964) [[Amazon]](https://www.amazon.com/dp/1801818967)

* The DevOps Career Handbook [[Packt]](https://www.packtpub.com/product/the-devops-career-handbook/9781803230948?utm_source=github&utm_medium=repository&utm_campaign=9781803230948) [[Amazon]](https://www.amazon.com/dp/1803230940)

## Errata
 * Page xxi (Under to get the most out of this book): **gPRC(https://grpc.io)** _should be_ **gRPC(https://grpc.io)**
 * Page 29 (Under Returning multiple values and named results): **func divide(num, div int) (res, rem int) { result = num / div remainder = num % div return res, rem }** _should be_ **func divide(num, div int) (res, rem int) {res = num / div rem = num % div return res, rem
}**
 * Page 74, Third paragraph **we will spin off 10 goroutines to add a number to a sum value** _should be_ **we will spin off 100 goroutines to add a number to a sum value**
 * Page 77, Code snippet: `if ctx.Err() != nil { return nil, err }` _should be_ ` if err := ctx.Err() != nil { return nil, err }`

## Get to Know the Authors
**John Doak**
is the principal manager of Layer 1 Reliability Engineering at Microsoft. John led the development of the Azure Data Explorer and Microsoft Authentication Library Go SDKs. Previously, he was a Staff Site Reliability Engineer at Google. As part of network engineering, he created many of their first network automation systems. John led the migration of that group from Python to Go, developing Go training classes that have been taught around the world. He was a pivotal figure in transforming the network team to a network/systems group that integrated with SRE. Prior to that, he worked for Lucasfilm in video games and film. You can find his musings on Go/SRE topics and his Go classes on the web.

**David Justice**
is the principal software engineer lead for the Azure K8s infrastructure and Steel Thread teams, which maintain a variety of CNCF and Bytecode Alliance projects. He is a maintainer of the Cluster API Provider Azure and a contributor to the Cluster API. Prior to that, David was the technical assistant to the Azure CTO, where he was responsible for Azure cross-group technical strategy and architecture. Early on at Microsoft, he was a program manager leading Azure SDKs and CLIs, where he transitioned all Azure services to describe them using OpenAPI specifications in GitHub and established automations to generate Azure reference docs, SDKs, and CLIs. Prior to working at Microsoft, David was the CTO of a mobile CI/CD SaaS called CISimple.
### Download a free PDF

 <i>If you have already purchased a print or Kindle version of this book, you can get a DRM-free PDF version at no cost.<br>Simply click on the link to claim your free PDF.</i>
<p align="center"> <a href="https://packt.link/free-ebook/9781801818896">https://packt.link/free-ebook/9781801818896 </a> </p>
