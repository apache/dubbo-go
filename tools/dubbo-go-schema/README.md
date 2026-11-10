# dubbo-go-schema

English | [中文](README_CN.md)

`dubbo-go-schema` provides the JSON Schema used by editors and YAML language servers to validate and complete Apache dubbo-go configuration files such as `dubbogo.yaml`, `dubbo.yaml`, and `application.yaml`.

The schema is maintained from the current `global/*_config.go` YAML tags and common dubbo-go configuration examples.

## Files

- `dubbo-go.json`: JSON Schema for dubbo-go YAML configuration.
- `application.yaml`: example configuration that exercises common provider, consumer, Triple, metrics, tracing, logger, and shutdown settings.
- `images/`: screenshots used by this README.

## VS Code

Install a YAML language server extension, such as the Red Hat YAML extension, then add this to `settings.json`:

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/apache/dubbo-go/develop/tools/dubbo-go-schema/dubbo-go.json": [
      "dubbo.yaml",
      "dubbogo.yaml",
      "application.yaml"
    ]
  }
}
```

When testing local schema changes, point VS Code at the checked-out schema file instead:

```json
{
  "yaml.schemas": {
    "./tools/dubbo-go-schema/dubbo-go.json": [
      "dubbo.yaml",
      "dubbogo.yaml",
      "application.yaml"
    ]
  }
}
```

## IntelliJ IDEA / GoLand

Open `Settings | Languages & Frameworks | Schemas and DTDs | JSON Schema Mappings`, add `dubbo-go.json`, and map it to the config file names you use.

## Validate Locally

At minimum, verify that the schema is valid JSON:

```bash
node -e "JSON.parse(require('fs').readFileSync('tools/dubbo-go-schema/dubbo-go.json', 'utf8'))"
```

For editor validation, open `application.yaml` in VS Code or GoLand with the local schema mapping enabled. The schema expects dubbo-go v3 style configuration under the `dubbo:` root key.

## Screenshots

![GoLand schema mapping](images/img.png)

![VS Code YAML schema mapping](images/vs-code.png)

![Completion result](images/img_1.png)
