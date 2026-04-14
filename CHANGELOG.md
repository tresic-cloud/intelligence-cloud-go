# Changelog

All notable changes to this project will be documented in this file.

This changelog is auto-generated from conventional commits by
[release-please](https://github.com/googleapis/release-please).
Manual edits between releases will be overwritten.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 1.0.0 (2026-04-14)


### Features

* **A-10:** add typed error hierarchy with sentinels and concrete types ([1ed84e4](https://github.com/tresic-cloud/intelligence-cloud-go/commit/1ed84e47937f078fffc4e1c7347ac8cf65b023cf))
* **A-12:** add RetryPolicy and JitterStrategy ([4d665c7](https://github.com/tresic-cloud/intelligence-cloud-go/commit/4d665c71241ad5ebe325321256e17442f8193a00))
* **A-14:** add CallOption, ListOption, and shared config structs ([5ea2445](https://github.com/tresic-cloud/intelligence-cloud-go/commit/5ea24455922f901c5a4b984790c224062c53c0e8))
* **A-18:** add generic Iterator[T] with PageInfo ([5755f2a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/5755f2a5e5f96b72ebcc9f2095abe173cb01e5e8))
* **A-21:** add Client struct and NewClient with functional options ([ec20da9](https://github.com/tresic-cloud/intelligence-cloud-go/commit/ec20da93b51bfe54f2048f42e1200088b1e419ec))
* **A-24:** add telemetry helpers -- span name, attrs, redaction, log handler ([8aa77fc](https://github.com/tresic-cloud/intelligence-cloud-go/commit/8aa77fc6592d4a465d965c70ea680a9fdd6a5869))
* **A-26:** wire default User-Agent from internal/version package ([d8baa0d](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d8baa0d58f5eba3dafda621c725c7d827b2e8361))
* **A-2:** add Token struct and CredentialProvider interface ([4138529](https://github.com/tresic-cloud/intelligence-cloud-go/commit/41385292873108937c1f1da7957fd8614ef4191b))
* **A-4:** add StaticToken credential provider ([e99f8f1](https://github.com/tresic-cloud/intelligence-cloud-go/commit/e99f8f1bdef64141debd37c2cd7d4bef83b7b19b))
* **A-6:** add RefreshFunc credential provider with expiry-aware cache ([ea5891b](https://github.com/tresic-cloud/intelligence-cloud-go/commit/ea5891b4fc2e3e6a89cab47ca7e5ce71f70e4526))
* **B-10:** implement MeService ([42e86df](https://github.com/tresic-cloud/intelligence-cloud-go/commit/42e86dff7061ff043eb95bd064f61c9bb77854bf))
* **B-12:** implement ResellerService ([d38c343](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d38c3430be406e7cdede91be8b474899d3cecc42))
* **B-14:** implement AuthService.Login with OAuth error envelope handling ([d50b54a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d50b54a1833e3d15ec63dcba4cff342382bc01de))
* **B-16:** implement AuditLogService.ListForReseller ([ccb6b65](https://github.com/tresic-cloud/intelligence-cloud-go/commit/ccb6b657ad2b59c866dd5970c2c3cb2f740eaf28))
* **B-18:** implement ConnectorService with location and company scopes ([dcbfbbd](https://github.com/tresic-cloud/intelligence-cloud-go/commit/dcbfbbd9bc4b4899ef6eea116a6eebc539bac311))
* **B-20:** implement ProductService.List ([e8a9d4e](https://github.com/tresic-cloud/intelligence-cloud-go/commit/e8a9d4eb1a3ca9d4613415f571a2bb410b73c332))
* **B-22:** implement UserService.PatchCompany ([cea0b6a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/cea0b6a91bd7be132d95581cd2e0df91244f0e78))
* **C-11:** add Transport RoundTripper composing auth, retry, OTel, slog, redaction, error mapping ([7a570af](https://github.com/tresic-cloud/intelligence-cloud-go/commit/7a570afd2f28ca2cfce5fa52270f41549fec2965))
* **C-16:** add operation-context plumbing ([dd4c7bd](https://github.com/tresic-cloud/intelligence-cloud-go/commit/dd4c7bd3ddc11176a82ef7527c52da0831f2681b))
* **C-2:** add transport redaction helpers ([c80ce67](https://github.com/tresic-cloud/intelligence-cloud-go/commit/c80ce6755703df2f31ebcef04f013ec56228874f))
* **C-4:** add retry decision function with Retry-After parsing ([054eaeb](https://github.com/tresic-cloud/intelligence-cloud-go/commit/054eaeb5fe2a01a72d11a87af47fa50565567da6))
* **D-12:** implement main entry, root cmd, and version subcommand ([98a4f10](https://github.com/tresic-cloud/intelligence-cloud-go/commit/98a4f10958f5e7072fedcae5db1e408fa3637ffb))
* **D-14:** implement tui/BuildClient ([7c49fcb](https://github.com/tresic-cloud/intelligence-cloud-go/commit/7c49fcbdbc295c23c9c85d93ef77fae209f18927))
* **D-18:** implement profile subcommand tree ([708686b](https://github.com/tresic-cloud/intelligence-cloud-go/commit/708686bca6fee13019add7ce547193aed547a437))
* **D-20:** implement resellers CLI subcommand with destructive preview ([efbb5be](https://github.com/tresic-cloud/intelligence-cloud-go/commit/efbb5be13a9f6791da86b7371745afaa3902bcd5))
* **D-21:** implement me CLI subcommand ([c1f9dc8](https://github.com/tresic-cloud/intelligence-cloud-go/commit/c1f9dc89d27962acac255b3f1972a77e8528646c))
* **D-26:** implement completion subcommand for bash/zsh/fish/powershell ([7b15025](https://github.com/tresic-cloud/intelligence-cloud-go/commit/7b15025214ac941c5e181ae27fe42390ae840f1c))
* **D-2:** implement SecretStore with keychain and file backends ([eff188a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/eff188a1b74a9fc6dee621e620855086ba7a12b4))
* **D-31:** add SIGINT handling for graceful shutdown ([94b51e0](https://github.com/tresic-cloud/intelligence-cloud-go/commit/94b51e016fe62136621ebf671cd441a8a661e007))
* **D-4:** implement Profile store with 0600 config.yaml ([6a4cdf9](https://github.com/tresic-cloud/intelligence-cloud-go/commit/6a4cdf9567f22936cd3940d3b05fae0da85081e7))
* **D-6:** implement JSON, table, and error formatters with exit-code mapping ([f1b6196](https://github.com/tresic-cloud/intelligence-cloud-go/commit/f1b61961fd76406b9ce383420ced4b98a3f18d54))
* **D-9:** implement DestructivePreview and confirmation prompt ([6e0745d](https://github.com/tresic-cloud/intelligence-cloud-go/commit/6e0745dca5e4d8fc2c17e356b446fa75a4b0eef5))
* **E-24:** add internal/version package with ldflags-injected version ([7407047](https://github.com/tresic-cloud/intelligence-cloud-go/commit/7407047d3ab62f4a2f117023ef3338306bd3dcff))
* wire shell completion subcommand into root cmd ([61a241d](https://github.com/tresic-cloud/intelligence-cloud-go/commit/61a241d25f41f9fbb98bd0514f104ab01aaa3d09))


### Bug Fixes

* pin olekukonko/tablewriter to v0.0.5 (pre-v1 stable API) ([96d7fe7](https://github.com/tresic-cloud/intelligence-cloud-go/commit/96d7fe72af674fbcaccddfc9b81fad8ace130319))
* rebuild go.mod/go.sum after Wave 3 merge-conflict resolution ([78b01cc](https://github.com/tresic-cloud/intelligence-cloud-go/commit/78b01cc751ca2dfd15fcbedf8a9a185141755bec))
* resolve AuditLogEntry duplicate declaration from Wave 2 merge ([4b346a0](https://github.com/tresic-cloud/intelligence-cloud-go/commit/4b346a07bed54c8b798443c90cb6bf1e5358327f))

## [Unreleased]
