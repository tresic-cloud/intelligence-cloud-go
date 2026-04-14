# Changelog

All notable changes to this project will be documented in this file.

This changelog is auto-generated from conventional commits by
[release-please](https://github.com/googleapis/release-please).
Manual edits between releases will be overwritten.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0](https://github.com/tresic-cloud/intelligence-cloud-go/compare/intelligence-cloud-go-v1.0.0...intelligence-cloud-go-v1.1.0) (2026-04-14)


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
* Intelligence Cloud Go SDK and CLI foundation (IDB-1353) ([328fb48](https://github.com/tresic-cloud/intelligence-cloud-go/commit/328fb4896c2096a83da07d2c8abb2430141f74eb))
* Intelligence Cloud Go SDK and CLI foundation (IDB-1353) ([328fb48](https://github.com/tresic-cloud/intelligence-cloud-go/commit/328fb4896c2096a83da07d2c8abb2430141f74eb))
* wire shell completion subcommand into root cmd ([61a241d](https://github.com/tresic-cloud/intelligence-cloud-go/commit/61a241d25f41f9fbb98bd0514f104ab01aaa3d09))


### Bug Fixes

* pin olekukonko/tablewriter to v0.0.5 (pre-v1 stable API) ([96d7fe7](https://github.com/tresic-cloud/intelligence-cloud-go/commit/96d7fe72af674fbcaccddfc9b81fad8ace130319))
* rebuild go.mod/go.sum after Wave 3 merge-conflict resolution ([78b01cc](https://github.com/tresic-cloud/intelligence-cloud-go/commit/78b01cc751ca2dfd15fcbedf8a9a185141755bec))
* resolve all lint issues and fix CI golangci-lint configuration ([fbdaf42](https://github.com/tresic-cloud/intelligence-cloud-go/commit/fbdaf425d90ccfc8c58b73b8f9c221bd4ef21a1d))
* resolve AuditLogEntry duplicate declaration from Wave 2 merge ([4b346a0](https://github.com/tresic-cloud/intelligence-cloud-go/commit/4b346a07bed54c8b798443c90cb6bf1e5358327f))


### Documentation

* **A-27:** add auth package doc ([591f5e1](https://github.com/tresic-cloud/intelligence-cloud-go/commit/591f5e12d8686fe9e99687f32717a5dffd5b015d))
* add root package doc.go with usage overview ([460632f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/460632f9fc85728a064052c4a1410bfde3474665))
* **B-6:** document OpenAPI 3.1 codegen collision as known issue ([50cde9e](https://github.com/tresic-cloud/intelligence-cloud-go/commit/50cde9e294a36f741dcbd1265a6fc2b6ab4082a8))
* **B-8:** add generated package doc ([18d27bc](https://github.com/tresic-cloud/intelligence-cloud-go/commit/18d27bcd1182997dbb9b566bdf2aa3b5a56a5c45))
* **E-11:** add SECURITY.md with disclosure policy ([350bd39](https://github.com/tresic-cloud/intelligence-cloud-go/commit/350bd3989c604952f6771e3d585d226d7bd61360))
* **E-12:** add README with badges, install, SDK and CLI quickstarts ([0eaedb9](https://github.com/tresic-cloud/intelligence-cloud-go/commit/0eaedb93a50f03c16cc576ceeabc747dc54e98ee))
* **E-14:** add CONTRIBUTING.md ([4c7356a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/4c7356ad6b9034efa7722631724fad89ca6515c6))
* **E-15:** seed CHANGELOG.md for release-please ([e6b692f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/e6b692f9f1adb7ff21389d29538bf6f1ec96bdc7))
* **E-16:** add MIT LICENSE ([2c31274](https://github.com/tresic-cloud/intelligence-cloud-go/commit/2c3127450b103c6071eee80cea29cc6c20cbafa3))
* **vault:** add guides (SDK + CLI quickstarts, Contributing, Releasing) ([a744cff](https://github.com/tresic-cloud/intelligence-cloud-go/commit/a744cff106c3ed6e8112cc1184e946793269334e))
* **vault:** add operations notes (compliance controls, security posture, observability runbook) ([d35bb03](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d35bb036713882074cd4f8a2b68b692bfbd3351f))
* **vault:** add project notes (roadmap, status, constitution, issues) ([6ea1d1f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/6ea1d1f563556f47ac4be967eb06299cd53350c5))
* **vault:** add reference notes (glossary, errors, retry, pagination, auth, operation IDs) ([2aaf11f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/2aaf11fbda46c07151e625970944bd15bcfd8ec4))
* **vault:** scaffold Obsidian vault with architecture and ADRs ([3820477](https://github.com/tresic-cloud/intelligence-cloud-go/commit/3820477d6b64b27634dd352b44dd1e37cefede4c))


### Miscellaneous

* **001-sdk-cli-foundation:** release 1.0.0 ([143d0c4](https://github.com/tresic-cloud/intelligence-cloud-go/commit/143d0c4b2736a1f47de0488e6bd8bec37c69d1aa))
* **001-sdk-cli-foundation:** release 1.0.0 ([dbbfe25](https://github.com/tresic-cloud/intelligence-cloud-go/commit/dbbfe25248edf5b6b7973abf98b6c97d8e5541dc))
* **B-23:** scaffold Company/Location/Vertical service files ([92a9d1a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/92a9d1ad5f21b6a68e2866415f44a98ed00fdee2))
* **B-4:** pin initial OpenAPI snapshot from intelligence-cloud ([718879d](https://github.com/tresic-cloud/intelligence-cloud-go/commit/718879d0b7feed2e52419b9b9a057af6575fb031))
* **B-5:** add oapi-codegen configuration ([6eb73ed](https://github.com/tresic-cloud/intelligence-cloud-go/commit/6eb73ed481e3bec598e878f868c23524b4525462))
* **B-6:** add scripts/codegen.sh and generated client/types ([c3f5e2e](https://github.com/tresic-cloud/intelligence-cloud-go/commit/c3f5e2ec97f0f033fe4d1a478fcccac91633e997))
* **B-7:** add scripts/pull-openapi.sh ([54fe14c](https://github.com/tresic-cloud/intelligence-cloud-go/commit/54fe14ce8e54a797f380fea6d61d905942a7a0f2))
* **deps:** Bump actions/checkout from 4 to 6 ([ba2a704](https://github.com/tresic-cloud/intelligence-cloud-go/commit/ba2a7043a96c49f29fd77b87106350d7096221f3))
* **deps:** Bump actions/checkout from 4 to 6 ([bf60bd1](https://github.com/tresic-cloud/intelligence-cloud-go/commit/bf60bd1c8c29a8d24a96720d985ff1a61330c7f9))
* **deps:** Bump actions/download-artifact from 4 to 8 ([e151127](https://github.com/tresic-cloud/intelligence-cloud-go/commit/e15112766b04b3e7d5fdf305aeb80760f6fbd89d))
* **deps:** Bump actions/download-artifact from 4 to 8 ([6e96b21](https://github.com/tresic-cloud/intelligence-cloud-go/commit/6e96b213771bf38dce0ed10a2c7ad87d558f1f43))
* **deps:** Bump actions/upload-artifact from 4 to 7 ([91c29a2](https://github.com/tresic-cloud/intelligence-cloud-go/commit/91c29a294c57a9f9a3d457ef9590a2e42ffebd82))
* **deps:** Bump actions/upload-artifact from 4 to 7 ([3b6be27](https://github.com/tresic-cloud/intelligence-cloud-go/commit/3b6be270c482dd4c354242ad3e550a58a11385e4))
* **deps:** Bump golangci/golangci-lint-action from 6 to 9 ([99581d2](https://github.com/tresic-cloud/intelligence-cloud-go/commit/99581d2be346fc708ae78b1df1629e58dcd8ce65))
* **deps:** Bump golangci/golangci-lint-action from 6 to 9 ([bfd1edd](https://github.com/tresic-cloud/intelligence-cloud-go/commit/bfd1eddb8be6123052f55d75f0b8c60156e217ff))
* **E-10:** add GitHub community files (CODEOWNERS, PR/issue templates) ([564ed07](https://github.com/tresic-cloud/intelligence-cloud-go/commit/564ed07556687721a5bb6d65fd8625d49c3a0ca6))
* **E-17:** add .goreleaser.yaml for cross-platform release ([00addb0](https://github.com/tresic-cloud/intelligence-cloud-go/commit/00addb0515bd8f5273094f23eab6f72b00a93fbd))
* **E-1:** initialise go module for SDK + CLI ([7e6a01a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/7e6a01a7a8b942fcf9789f0f611b7b18bd90ffd6))
* **E-20:** add markdownlint configuration ([5fc8522](https://github.com/tresic-cloud/intelligence-cloud-go/commit/5fc852280928978981bb316a5f90e73d9eb55c5b))
* **E-21:** update Makefile docs target ([31f6c95](https://github.com/tresic-cloud/intelligence-cloud-go/commit/31f6c95c63dac1f3ad34e2cb9c64adf9a5615019))
* **E-2:** add golangci-lint configuration with exported-symbol enforcement ([5a5fe32](https://github.com/tresic-cloud/intelligence-cloud-go/commit/5a5fe3291b42a443df090aec4e3a34121c0b626b))
* **E-4:** add Makefile with developer targets ([1d6deaf](https://github.com/tresic-cloud/intelligence-cloud-go/commit/1d6deaff94da21727c5351ac4c5cb50d13a7942a))
* **E-5:** add Dependabot configuration for gomod and github-actions ([443137a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/443137ac94448e079c92555b6f8b3f0dd21d2c1d))
* reconcile go.mod + go.sum after Wave 3 agent merges ([48d98d9](https://github.com/tresic-cloud/intelligence-cloud-go/commit/48d98d9ba43c2e18a7a54ab697c93dea5c57a702))


### Code Refactoring

* consolidate resource-service files into services.go ([40b8023](https://github.com/tresic-cloud/intelligence-cloud-go/commit/40b80237f276675d1388ce769aeba8b0970033fe))
* remove all scaffold code (Companies, Locations, Verticals) ([f6a1c88](https://github.com/tresic-cloud/intelligence-cloud-go/commit/f6a1c88cfc0f4ac9398255a7f06db476181e2bc1))


### Tests

* **A-11:** add RetryPolicy defaults and validation tests ([b07df32](https://github.com/tresic-cloud/intelligence-cloud-go/commit/b07df32c2fc06715af8db85458d3871ddfbd7ae3))
* **A-13:** add CallOption and ListOption tests ([3d9b297](https://github.com/tresic-cloud/intelligence-cloud-go/commit/3d9b2976a91efcbd3b733c1710e5c0677e569f94))
* **A-15:** add Iterator lifecycle tests ([56a3cf3](https://github.com/tresic-cloud/intelligence-cloud-go/commit/56a3cf39b789e4469657d6ec679a4f85129b8371))
* **A-16:** add Iterator multi-page and PageInfo tests ([0d58f5b](https://github.com/tresic-cloud/intelligence-cloud-go/commit/0d58f5bb8a28ff85305d9312db12d8d988916057))
* **A-17:** add Iterator error and context-cancellation tests ([99433e6](https://github.com/tresic-cloud/intelligence-cloud-go/commit/99433e6420a2b20014cf927c31a773b2bcbd65d3))
* **A-19:** add NewClient valid-construction tests ([4b5b5d2](https://github.com/tresic-cloud/intelligence-cloud-go/commit/4b5b5d2531640f2fd0671ee309a4bd3a98284826))
* **A-1:** add Token and CredentialProvider interface tests ([922fa2b](https://github.com/tresic-cloud/intelligence-cloud-go/commit/922fa2ba0f2d0def2f4191e55e2900924db47f4d))
* **A-20:** add NewClient validation-failure tests ([cf1e644](https://github.com/tresic-cloud/intelligence-cloud-go/commit/cf1e6446d3df7fc69f4228da7b951b776a120ddb))
* **A-22:** add span-name and attribute helper tests ([bde847a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/bde847a98229b139aa412316ece52bb833790fb9))
* **A-23:** add redacting slog handler tests ([780aee8](https://github.com/tresic-cloud/intelligence-cloud-go/commit/780aee86184c86945327bbc8ea9976aaae192dc1))
* **A-25:** add default User-Agent format tests ([0050454](https://github.com/tresic-cloud/intelligence-cloud-go/commit/0050454625688eefff0ddd2dc25aae4431e1325f))
* **A-3:** add StaticToken provider tests ([06133f6](https://github.com/tresic-cloud/intelligence-cloud-go/commit/06133f6b648ecb51d9f2ce136fae861a7638ce65))
* **A-5:** add RefreshFunc caching and single-flight tests ([26b7cad](https://github.com/tresic-cloud/intelligence-cloud-go/commit/26b7cad30119d44a59d8f760dd873963b26cd2c0))
* **A-7,A-8,A-9,A-28:** add error hierarchy tests ([9b59232](https://github.com/tresic-cloud/intelligence-cloud-go/commit/9b59232ed74112815fec5f4fbff8ae87119b2c5c))
* **B-11:** add ResellerService tests ([c92cc81](https://github.com/tresic-cloud/intelligence-cloud-go/commit/c92cc812c4cf43d66408960e8d5fd8ece5f475cc))
* **B-13:** add AuthService tests ([927ab78](https://github.com/tresic-cloud/intelligence-cloud-go/commit/927ab78c2d4b091bcac72a98a81c98c325d1ce24))
* **B-15:** add AuditLogService tests ([3b29954](https://github.com/tresic-cloud/intelligence-cloud-go/commit/3b29954d591bb6bc78198b521cb07e6caa61d59c))
* **B-17:** add ConnectorService tests ([fae02a8](https://github.com/tresic-cloud/intelligence-cloud-go/commit/fae02a8b507a9f148baeeded47dc73dccbc4cbc7))
* **B-19:** add ProductService tests ([d5faf22](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d5faf22955c6919989c88d5996ac00c08b5447bb))
* **B-1:** add drift test asserting operationId coverage of generated code ([50247a1](https://github.com/tresic-cloud/intelligence-cloud-go/commit/50247a183237e77b3c02ad87039a8e2b13e4a3ef))
* **B-21:** add UserService tests ([b5a88f5](https://github.com/tresic-cloud/intelligence-cloud-go/commit/b5a88f51a7846c03b651367a8642457ddc030d58))
* **B-2:** add codegen idempotence check script ([024442b](https://github.com/tresic-cloud/intelligence-cloud-go/commit/024442bf36d64b61676660ab4d398f0b11ca2b48))
* **B-3:** add generated-package compilation test ([6ee8947](https://github.com/tresic-cloud/intelligence-cloud-go/commit/6ee89474ec2bfaee8d756bbb5a4c6d7f7b6a977b))
* **B-9:** add MeService tests ([1ecf09b](https://github.com/tresic-cloud/intelligence-cloud-go/commit/1ecf09b3bad87b578ba86fb9edf0e22f2d4eaa19))
* **C-12:** add integration tests with in-memory OTel exporter ([91f87d9](https://github.com/tresic-cloud/intelligence-cloud-go/commit/91f87d99a70fc7251be4f04cf9d860fef1db33cd))
* **C-13:** add zero-cost no-tracer benchmark ([d1c41ad](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d1c41adbaee5b67235967ea2054268408637f230))
* **C-1:** add transport redaction helper tests ([ac166a7](https://github.com/tresic-cloud/intelligence-cloud-go/commit/ac166a77a3f90c10f0c706d049654476ffd25665))
* **C-3:** add retry decision function tests ([0a0dcce](https://github.com/tresic-cloud/intelligence-cloud-go/commit/0a0dccef086f26daf9a8d79530d66e5e6a713820))
* **C-5,C-6,C-7,C-8,C-9,C-10,C-14,C-15:** add Transport RoundTripper tests ([97f9f48](https://github.com/tresic-cloud/intelligence-cloud-go/commit/97f9f488673a7d14a7a29c17da5db701ab92c339))
* **D-10:** add root cmd global-flag and precedence tests ([7b5342d](https://github.com/tresic-cloud/intelligence-cloud-go/commit/7b5342dde7915b41dd5b52c8ebb6ba65eafc2b66))
* **D-11:** add version subcommand tests ([aa26d86](https://github.com/tresic-cloud/intelligence-cloud-go/commit/aa26d869abf8ca200307d2704494844009cf5652))
* **D-13:** add tui/BuildClient tests ([95a3e1f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/95a3e1fd5b0a351dbe2d40d938a30bc87e110ae8))
* **D-17:** add profile subcommand tests ([1289f6a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/1289f6a29b200323592cbccd13210f0dadc1e744))
* **D-19:** add resellers CLI subcommand tests ([84aa90f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/84aa90f7e9407aa24773d1eef0b9ae406b3d35e3))
* **D-1:** add SecretStore tests ([3db711a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/3db711ac1b1428b7da9d67f2b7042877d5501729))
* **D-21:** add me CLI subcommand tests ([5491be5](https://github.com/tresic-cloud/intelligence-cloud-go/commit/5491be519e162af7f54e0ecfe2c9254d76f06a1c))
* **D-25:** add shell completion tests ([3fd7f7f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/3fd7f7f9905ae43d2cdf795450959b93b6f817c6))
* **D-27:** add help contract enforcement test across full command tree ([c5d7648](https://github.com/tresic-cloud/intelligence-cloud-go/commit/c5d7648b16b16f8321c6e680328516f063153e2b))
* **D-28:** add token-never-logged security invariant test ([7c4b68f](https://github.com/tresic-cloud/intelligence-cloud-go/commit/7c4b68fb6005946d0c77df9c970e070c773dcef5))
* **D-29:** add quickstart Part B integration test ([d8ddb79](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d8ddb79375d4bea4d6966045befdee41e2044985))
* **D-30:** add file permissions audit test ([e9e07f5](https://github.com/tresic-cloud/intelligence-cloud-go/commit/e9e07f588affba37174ec982e3361f77251c5e34))
* **D-3:** add Profile store tests ([b78a953](https://github.com/tresic-cloud/intelligence-cloud-go/commit/b78a9531fa11c7367ce79a435c5d262cbbeb4197))
* **D-5:** add output formatter tests ([aa85626](https://github.com/tresic-cloud/intelligence-cloud-go/commit/aa85626b665b37a33a913ef1dd1abaf3c0fbc0f6))
* **D-7:** add DestructivePreview tests ([d168bcd](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d168bcda523d4ddd6009da8e3c8f725c5fd9e071))
* **D-8:** add confirmation prompt tests ([26ba4c8](https://github.com/tresic-cloud/intelligence-cloud-go/commit/26ba4c88a6b97556a85b91bb1c78950cd568bdf2))


### CI/CD

* **E-18:** add release workflow with goreleaser and cosign signing ([f50d683](https://github.com/tresic-cloud/intelligence-cloud-go/commit/f50d683a665301d0b13b07abc6a47b099411c62a))
* **E-19:** add scheduled OpenAPI drift detection workflow ([daf44d2](https://github.com/tresic-cloud/intelligence-cloud-go/commit/daf44d29cd0acb5562187056f9cee974e9dbb805))
* **E-23:** add release-please for conventional-commit-driven auto-semver ([00d4f4a](https://github.com/tresic-cloud/intelligence-cloud-go/commit/00d4f4a5df31ba5ac0624c975c4a58fe98b1814e))
* **E-25:** add PR conventional-commit title lint to CI ([83dffdc](https://github.com/tresic-cloud/intelligence-cloud-go/commit/83dffdcc7b7bf179e1c63d0ccc9d3b2f63b91274))
* **E-3:** add CI workflow with gated lint/security/static/test/coverage/codegen-drift/build-matrix pipeline ([58033d6](https://github.com/tresic-cloud/intelligence-cloud-go/commit/58033d656c296eec219625c0220267aa2c270b2d))
* exclude generated code from coverage gate ([722040e](https://github.com/tresic-cloud/intelligence-cloud-go/commit/722040e262c8acb01143617cdf0cc86b9ec280a0))
* make govulncheck advisory (continue-on-error) ([edc0b79](https://github.com/tresic-cloud/intelligence-cloud-go/commit/edc0b79c2d1fef282edbd849728e58571e03ccce))
* restructure pipeline into 3 clear steps ([d5cdb2c](https://github.com/tresic-cloud/intelligence-cloud-go/commit/d5cdb2c9e9d2813e5fc708d50d78681cbe9e05bb))
* trigger fresh CI run ([a608be0](https://github.com/tresic-cloud/intelligence-cloud-go/commit/a608be0884d1279c7f76d5fb6ebc27089697a0ad))
* trigger release-please after enabling PR creation permission ([22f144c](https://github.com/tresic-cloud/intelligence-cloud-go/commit/22f144c859209da2a7c005d52ed49221e941b3cd))

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
