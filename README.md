# Terraform Provider for Revos

This provider manages Revos resources, such as Cube Overlays.

## Installation

### 1. Configure the mirror

You can configure the provider mirror either globally or per-project.

#### Option A: Global configuration

Add to `~/.terraformrc` (macOS/Linux) or `%APPDATA%\terraform.rc` (Windows):

```hcl
provider_installation {
  network_mirror {
    url     = "https://revosai.github.io/terraform-provider-revos/"
    include = ["revosai/revos"]
  }
  direct {
    exclude = ["revosai/revos"]
  }
}
```

#### Option B: Project-local configuration (recommended)

Create a `mirror.tfrc` file in your Terraform project:

```hcl
provider_installation {
  network_mirror {
    url     = "https://revosai.github.io/terraform-provider-revos/"
    include = ["revosai/revos"]
  }
  direct {
    exclude = ["revosai/revos"]
  }
}
```

Then run Terraform with the config file:

```bash
# Set for single command
TF_CLI_CONFIG_FILE=./mirror.tfrc terraform init

# Or export for the session
export TF_CLI_CONFIG_FILE=./mirror.tfrc
terraform init
terraform plan
terraform apply
```

You can also add a Makefile for convenience:

```makefile
export TF_CLI_CONFIG_FILE := $(CURDIR)/mirror.tfrc

init:
	terraform init

plan:
	terraform plan

apply:
	terraform apply
```

### 2. Use the provider

```hcl
terraform {
  required_providers {
    revos = {
      source  = "revosai/revos"
      version = "0.1.0"
    }
  }
}
```

## Usage

### Provider Configuration

```hcl
provider "revos" {
  api_url = "https://api.revos.io" # Optional, or set REVOSAI_API_URL
  token   = "your-api-token"       # Required, or set REVOSAI_TOKEN
}
```

### Resource: `revos_overlay`

```hcl
resource "revos_overlay" "example" {
  name        = "my-overlay"
  description = "An example overlay"

  # Data must be a JSON string
  data = jsonencode({
    measures = {
      count = {
        type = "count"
      }
    }
  })
}
```

### Import

Overlays can be imported by ID or name:

```bash
terraform import revos_overlay.example overlay-id-here
terraform import revos_overlay.example overlay-name-here
```

## Development

### Requirements

- Terraform >= 1.0
- Go >= 1.21

### Building

```bash
go build -o terraform-provider-revos
```

### Local Testing

1. Build the provider
2. Create `dev_overrides.tfrc`:
   ```hcl
   provider_installation {
     dev_overrides {
       "registry.terraform.io/revosai/revos" = "/path/to/terraform-provider-revos"
     }
     direct {}
   }
   ```
3. Export: `export TF_CLI_CONFIG_FILE=$(pwd)/dev_overrides.tfrc`
4. Run `terraform plan`

### Running Tests

```bash
go test -v ./...
```

## Releasing

Releases are automated via GitHub Actions when a tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The release workflow will build binaries for all platforms and publish to GitHub Releases.
