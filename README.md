# Terraform Provider for Zeet

Zeet simplifies the deployment and management of applications and infrastructure on your cloud. With this Terraform
provider, you can automate and manage Zeet's powerful features through Terraform.

## Features

- **Deploy and Manage Zeet Projects**: Automate the creation and management of Zeet projects, including service
  containers, Helm releases, and Terraform modules.
- **Integrate with Terraform Ecosystem**: Seamlessly integrate with other Terraform native cloud modules.
- **Security & Compliance**: Ensure your deployments meet security and compliance standards.

### Supported Resources & Data

We currently support these entities; new features are being added, so feel free to request more.

| Entity Type | Terraform Resource | Terraform Data |
| ----------- | ------------------ | -------------- |
| Team        | ✅                 |                |
| Group       | ✅                 | ✅             |
| SubGroup    | ✅                 | ✅             |
| Blueprint   |                    | ✅             |
| Project     | ✅                 |                |

### Mapping to Project Types

| Zeet Project Type   | Blueprint Slug                  |
| ------------------- | ------------------------------- |
| Service Container   | zeet-kubernetes-container-app   |
| Job Container       | zeet-kubernetes-container-job   |
| Serverless          | Not Supported at the moment     |
| Terraform Module    | helm-chart                      |
| Helm Release        | terraform-module                |
| Kubernetes Manifest | kubernetes-manifest             |
| Database            | refers to database blueprint id |
| Custom Blueprint    | refers to your own blueprint id |

## Getting Started

To get started with the Terraform Provider for Zeet, ensure you have Terraform installed and then add the Zeet provider
to your Terraform configuration:

```hcl
terraform {
  required_providers {
    zeet = {
      source  = "zeet-dev/zeet"
      version = "~> 1.0.0"
    }
  }
}

provider "zeet" {
  # Configuration options
}
```

For detailed instructions on how to use this provider, including setup, configuration, and examples, refer to the
[Terraform Provider Documentation](https://registry.terraform.io/providers/zeet-dev/zeet/latest/docs).

## Contributing

We welcome contributions from the community! Whether it's adding new features, improving documentation, or reporting
bugs, please feel free to make a pull request or open an issue.

## Development Guide

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.20

### Building The Provider

1. Clone the repository.
2. Enter the repository directory.
3. Build the provider using the Go `install` command:

```shell
go install
```

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules). Please refer to the Go documentation for the
most up-to-date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then, commit the changes to `go.mod` and `go.sum`.

### Developing the Provider

To generate or update documentation, run `go generate`.

To run the full suite of Acceptance tests, which create real resources and often cost money, run `make testacc`.

```shell
make testacc
```

## License

This Terraform provider's source code is adapted from the official template
https://github.com/hashicorp/terraform-provider-scaffolding-framework
