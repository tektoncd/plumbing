# :cat: Catlin :cat:

Catlin is a command-line tool that Lints Tekton Resources and Catalogs.

It validates the resources on the basis the Tekton catalog structure defined in the [TEP][tep].

![](images/demo.gif)

## Commands

### Validate

This command validates
- If the resource is in valid path
- If the resource is a valid Tekton Resource
- If all mandatory fields are added to the resource file
- If all images used in Task Spec are tagged
- If platforms are specified in correct format
```
catlin validate <path-to-resource-file>
```

[tep]:https://github.com/tektoncd/community/blob/main/teps/0003-tekton-catalog-organization.md
