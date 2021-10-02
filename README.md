# pulumi-helm-chart-base

This repository contains boilerplate for creating a [Pulumi Component Package](
https://www.pulumi.com/docs/guides/pulumi-packages/) which wraps a Kubernetes [Helm Chart](https://helm.sh),
and gives it a strongly typed interface. This exposes the chart to the Pulumi's Infrastructure as Code tool in
multiple languages, including JavaScript, TypeScript, Python, Go, and C#, and adds compile-time type-checking
for chart parameters, built-in documentation, and more.
