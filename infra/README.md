# Overview

# Requirements

- Latest version of [Terraform](https://developer.hashicorp.com/terraform/install)
    - For configuration information, visit [Hashicorp's documentation](https://developer.hashicorp.com/terraform/tutorials/azure-get-started), or [Microsoft's documentation](https://learn.microsoft.com/en-us/azure/developer/terraform/)

# Usage

### Variables

In the directory with the Terraform files, create a `terraform.tfvars` file, and define the `prefix`(required) and any `tags`(optional).

### Deploying

To deploy the Azure infrastructure defined in the `main.tf` file, navigate in your terminal to the directory containing the Terraform files and run the below commands:

```bash
terraform init

terraform apply --auto-approve
```

To view the outputs defined in the `outputs.tf` file, run the below command:

```bash
terraform output
```