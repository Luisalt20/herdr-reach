{
  "schema": "gentle-ai.sdd-research/v1",
  "revision": 2,
  "change": "reach-diagnosis-core",
  "worktree": "/home/luisalt20/projects/close/herdr-reach",
  "artifact_store": "both",
  "outcome": "partial",
  "proposal_ready": false,
  "retrieved_at": "2026-09-14",
  "retained_intent": "R1a headless diagnosis core: verify the PRD external claims (cloudflared pin and service tokens, QUIC/http2, Dev Tunnels, macOS trust store, WSL2 vmIdleTimeout) before the slice-1 proposal depends on them.",
  "skill_resolution": "none",
  "scope_note": "Only the two carried locators were read (openspec/changes/reach-diagnosis-core/research.md and preproposal.md, plus Engram observations 793 and 792). explore.md was NOT read: the bounded scope refuses it. No skill path was injected and no skill file read was attempted, so skill_resolution is none. No subagents were launched; nothing was staged or committed.",
  "verbatim_note": "Quoted fragments from sources are rendered with single quotes instead of the source's double quotes so the document stays plain JSON; wording is otherwise verbatim.",
  "admission": {
    "documentation": {
      "selected": true,
      "declared_tools": ["fetch_content"],
      "observed_active_tools": ["fetch_content"],
      "status": "available",
      "evidence": "fetch_content executed and returned publisher pages for Cloudflare docs, Microsoft Learn, Microsoft devblogs and the Go project; original pages were fetched, not snippets."
    },
    "open-web": {
      "selected": true,
      "declared_required_tools": ["web_search", "source_check", "fetch_content", "get_search_content"],
      "observed_active_tools": ["web_search", "source_check", "fetch_content", "get_search_content"],
      "status": "partial",
      "evidence": "All four required tools were called and returned. web_search succeeded via the auto provider but one batch failed with an Exa 429 rate limit, and explicit brave/openai providers failed for missing keys. source_check ran three times but each returned 'unclear' (confidence 0.30) rather than support or contradiction, so its verdicts were NOT used as evidence; only fetched originals were. Class is recorded as partial because search and source-check coverage was degraded, not because a required tool was absent."
    },
    "selection_extensions": {
      "provided": false,
      "note": "No research_selection tools/extensions map was carried in this child context, so no --extension provenance was asserted for any route."
    },
    "unsupported_classes": [
      {
        "class": "generic-mcp-and-gateways",
        "status": "denied",
        "reason": "Not an approved evidence route; no evidence was drawn from MCP gateways, bash, or persistence tools."
      }
    ]
  },
  "failed_calls": [
    {
      "tool": "web_search",
      "input": "queries: cloudflared release notes 2026.6.0 service token; cloudflared TUNNEL_TRANSPORT_PROTOCOL quic http2; cloudflare tunnel post-quantum",
      "error": "Auto provider (Exa) returned HTTP 429 free-tier rate limit for that batch; results unavailable.",
      "recovery": "Re-ran the same angles on the auto provider in a later batch and fetched the publisher pages directly."
    },
    {
      "tool": "web_search",
      "input": "provider=brave, TUNNEL_TRANSPORT_PROTOCOL queries",
      "error": "Brave Search API key not configured in this environment.",
      "recovery": "Used the auto provider and direct documentation fetches instead."
    },
    {
      "tool": "web_search",
      "input": "provider=openai, transport protocol / post-quantum queries",
      "error": "OpenAI web search unavailable (no Codex session or API key).",
      "recovery": "Used the auto provider and direct documentation fetches instead."
    },
    {
      "tool": "source_check",
      "input": "claim: cloudflared 2026.6.0+ ignores service tokens for access ssh/tcp (issue 1673)",
      "error": "Verdict 'unclear', confidence 0.30; no support or contradiction markers extracted.",
      "recovery": "Treated as non-evidence; relied on the fetched issue body and GitHub API JSON."
    },
    {
      "tool": "source_check",
      "input": "claim: Microsoft documents vmIdleTimeout and the child-of-init shutdown rule",
      "error": "Verdict 'unclear', confidence 0.30; no support or contradiction markers extracted.",
      "recovery": "Treated as non-evidence; relied on the fetched Microsoft Learn page text."
    },
    {
      "tool": "source_check",
      "input": "claim: Dev Tunnels forwards raw TCP and authenticates headlessly with a token",
      "error": "Verdict 'unclear', confidence 0.30; no support or contradiction markers extracted.",
      "recovery": "Treated as non-evidence; relied on the fetched Microsoft Learn CLI and security pages."
    },
    {
      "tool": "fetch_content",
      "input": "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/configure-tunnels/tunnel-with-self-hosted-infrastructure/",
      "error": "HTTP 404 - page moved or renamed.",
      "recovery": "Used the current Cloudflare One run-parameters and tunnel-with-firewall pages instead."
    },
    {
      "tool": "fetch_content",
      "input": "mode=answer over the tunnel-with-firewall page",
      "error": "Provider error MissingSessionID (400) from the answer backend.",
      "recovery": "Re-fetched the same URL in readable mode and read the page text directly."
    }
  ],
  "sources": [
    {
      "id": "s1",
      "publisher": "Cloudflare (cloudflare/cloudflared issue tracker)",
      "kind": "issue-tracker",
      "url": "https://github.com/cloudflare/cloudflared/issues/1673",
      "url_secondary": "https://api.github.com/repos/cloudflare/cloudflared/issues/1673",
      "title": "access tcp/ssh: service token ignored, browser auth attempted per-connection, TCP banner not delivered (2026.6.0 regression from 2026.5.1)",
      "version_or_date": "issue created 2026-06-10T18:09:37Z; reporter environment: cloudflared client 2026.6.0 (built 2026-06-08T18:16:09Z) on macOS arm64, daemon 2026.5.1",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": [
        "fetch_content read: https://github.com/cloudflare/cloudflared/issues/1673",
        "fetch_content read: https://api.github.com/repos/cloudflare/cloudflared/issues/1673",
        "web_search auto: cloudflared access ssh service token ignored browser login issue 1673 github"
      ],
      "excerpt": "API JSON: state=open, created_at=2026-06-10T18:09:37Z, updated_at=2026-06-10T18:09:37Z, closed_at=null, comments=0, labels=[], assignees=[]. Body: 'cloudflared access tcp (and access ssh) in 2026.6.0 ignores all headless auth credentials (service-token-id/secret flags, $TUNNEL_SERVICE_TOKEN_ID/$SECRET env vars, -H header injection) and falls through to interactive browser auth on every new TCP connection. SSH banner never reaches the client; connection times out.' and 'This is a regression from 2026.5.1, which honored service tokens correctly and never attempted browser auth when credentials were provided.' Workaround stated: 'Downgrade to cloudflared 2026.5.1. No client-side workaround in 2026.6.0 has been found.'",
      "evidence_limits": "Single unreproduced report by a non-member author (author_association NONE). No Cloudflare maintainer comment, no label, no linked fix, no reproduction by a second party."
    },
    {
      "id": "s2",
      "publisher": "Cloudflare (cloudflare/cloudflared releases)",
      "kind": "release-page",
      "url": "https://github.com/cloudflare/cloudflared/releases/tag/2026.5.1",
      "title": "Release 2026.5.1",
      "version_or_date": "tag 2026.5.1",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: https://github.com/cloudflare/cloudflared/releases/tag/2026.5.1"],
      "excerpt": "SHA256 Checksums block lists 27 published artifacts, including: 'cloudflared-darwin-arm64.tgz: 086dd6628a3608eba7327af1092abcc52b10994524112ae49bf356ec49bef8c4', 'cloudflared-darwin-amd64.tgz: c232c2be1fda76ffb5dff55874c4d4e3d3137bd426e3e73d539c3ecbe20526a2', 'cloudflared-linux-amd64: 3c6a5ba995a258dbe90f98e5fdb2c2620b7be72c3ca761614f6eb52aee252cea', 'cloudflared-windows-amd64.exe: 8b97b5af442651e07c52caa9c79c6f60032bc10b675c2b36dd11c7690d9942e3'.",
      "evidence_limits": "The fetched excerpt covers the checksum block only; the publish timestamp on the page was not inside the fetched text."
    },
    {
      "id": "s3",
      "publisher": "Cloudflare (cloudflare/cloudflared repository)",
      "kind": "changelog",
      "url": "https://raw.githubusercontent.com/cloudflare/cloudflared/master/RELEASE_NOTES",
      "title": "RELEASE_NOTES (master)",
      "version_or_date": "entries through the 2026.8.x line (master at retrieval time)",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: https://raw.githubusercontent.com/cloudflare/cloudflared/master/RELEASE_NOTES"],
      "excerpt": "Release headers present include '2026.5.1 - 2026-05-22 fix: Bump go to 1.26.3 ...', '2026.6.0 - 2026-06-08 TUN-10558: Bump go to v1.24.4 ...', '2026.6.1 - 2026-06-18 TUN-10630: Fix precheck protocol override', '2026.7.0 - 2026-07-08 chore: Bump go-chi to version 5.3.1', '2026.8.0 - 2026-08-12 AUTH-9041 Generate ephemeral keypair for login token transfer', '2026.8.3 - 2026-08-28 ...'. Text search for 'service token' and 'access tcp' returned no matches anywhere in the file; the most recent entry mentioning 'access ssh' is '2025-03-17 TUN-9101: Don't ignore errors on cloudflared access ssh'.",
      "evidence_limits": "Absence of a changelog entry is not proof that the reported behaviour was not fixed in another commit."
    },
    {
      "id": "s4",
      "publisher": "Cloudflare (Cloudflare One / Tunnel documentation)",
      "kind": "official-documentation",
      "url": "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/configure-tunnels/tunnel-run-parameters/",
      "title": "Tunnel run parameters",
      "version_or_date": "current docs page at retrieval time (2026-09-14)",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": [
        "fetch_content read: the run-parameters page",
        "web_search auto: cloudflared --protocol quic http2 TUNNEL_TRANSPORT_PROTOCOL tunnel run parameters"
      ],
      "excerpt": "On post-quantum: 'By default, Cloudflare Tunnel connections over quic are encrypted using post-quantum cryptography (PQC) but will fall back to non-PQ if there are issues connecting. If the --post-quantum flag is provided, quic connections are only allowed to use PQ key agreements, with no fallback to non-PQ. Post-quantum key agreements are not supported when using http2 protocol.' On protocol: 'cloudflared tunnel --protocol <VALUE> run <UUID or NAME>', default 'auto', environment variable 'TUNNEL_TRANSPORT_PROTOCOL', 'Specifies the protocol used to establish a connection between cloudflared and the Cloudflare global network. Available values are auto, http2, and quic. The auto value will automatically configure the quic protocol. If cloudflared is unable to establish UDP connections, it will fallback to using the http2 protocol.'",
      "evidence_limits": "The page documents values and fallback behaviour; it does not contain the literal phrase 'forces HTTP/2'."
    },
    {
      "id": "s5",
      "publisher": "Cloudflare (Cloudflare One / Tunnel documentation)",
      "kind": "official-documentation",
      "url": "https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/tunnel-with-firewall/",
      "title": "Tunnel with firewall",
      "version_or_date": "current docs page at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: the tunnel-with-firewall page"],
      "excerpt": "'cloudflared connects to Cloudflare's global network on port 7844. To use Cloudflare Tunnel, your firewall must allow outbound connections to the following destinations on port 7844 (via UDP if using the quic protocol or TCP if using the http2 protocol).' and 'Ensure port 7844 is allowed for both TCP and UDP protocols (for http2 and quic).' SNI hostnames: 'quic.cftunnel.com ... 7844 ... UDP (quic)', 'h2.cftunnel.com ... 7844 ... TCP (http2)'.",
      "evidence_limits": "None material for the transport claim."
    },
    {
      "id": "s6",
      "publisher": "Microsoft (Microsoft Learn, dev tunnels)",
      "kind": "official-documentation",
      "url": "https://learn.microsoft.com/en-us/azure/developer/dev-tunnels/cli-commands",
      "title": "Dev tunnels command-line reference",
      "version_or_date": "current docs page at retrieval time (devtunnel CLI marked preview)",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": [
        "fetch_content read: the dev tunnels CLI reference page",
        "web_search auto: Microsoft Dev Tunnels forward raw TCP port tunnel SSH client; devtunnel CLI authentication token non-interactive Microsoft Learn"
      ],
      "excerpt": "On forwarding: 'Instead of having a client browser or application connect directly to a dev tunnel relay URI, the CLI may be used to forward connections from a port on the client to a dev tunnel port. The client may also need to log in, if the dev tunnel doesn't allow anonymous access.' Command 'devtunnel connect TUNNELID'; sample output 'Connected to tunnel: l3rs99qw' / 'SSH: Forwarding from 127.0.0.1:3000 to host port 3000.' / 'SSH: Forwarding from [::1]:3000 to host port 3000.' with the note '(The SSH prefix is because the dev tunnel service builds on the standard SSH protocol for port-forwarding.)' and 'Now, the server that was shared on the host's port 3000 is available at localhost:3000 on the client, using either IPv4 or IPv6.' On port protocol: 'When creating a port, the protocol may optionally be specified, if auto-detection doesn't work properly. Current options are http, https or auto (default).' On login: 'devtunnel user login' [...] 'Login with a Microsoft or GitHub account'; 'devtunnel user login -d' [...] 'with device code login, if local interactive browser login isn't possible'. On tokens: 'devtunnel token TUNNELID --scopes connect' issues 'a connect access token for a dev tunnel that can be shared to provide temporarily access to the dev tunnel'.",
      "evidence_limits": "The page's examples are HTTP/web oriented; it contains no SSH-client scenario and no tcp value for a port protocol."
    },
    {
      "id": "s7",
      "publisher": "Microsoft (Microsoft Learn, dev tunnels security)",
      "kind": "official-documentation",
      "url": "https://learn.microsoft.com/en-us/azure/developer/dev-tunnels/security",
      "title": "Dev tunnels security",
      "version_or_date": "current docs page at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: the dev tunnels security page"],
      "excerpt": "On auth defaults: 'By default, hosting and connecting to a tunnel requires authentication with the same Microsoft, Microsoft Entra ID, or GitHub account that created the tunnel.' On headless access: 'The CLI can also be used to request access tokens that grant limited access to anyone holding the token (use devtunnel token).' Four token types are documented: client, host, manage ports and management access tokens. 'The tokens expire after some time (currently 24 hours). Tokens can only be refreshed using an actual user identity that has manage-scope access to the tunnel (not just a management access token). Most CLI commands can accept a --access-token argument with an appropriate token as an alternative to logging in.' Web clients: 'X-Tunnel-Authorization: tunnel <TOKEN>' with the Tip 'This is useful for non-interactive clients as it allows them to access tunnels without requiring anonymous access to be enabled.' Anonymous access: 'an allow-anonymous Access control entry (ACE) can be added (use --allow-anonymous)'.",
      "evidence_limits": "The words 'service principal' do not appear on the page; neither do any managed-identity flags."
    },
    {
      "id": "s8",
      "publisher": "Microsoft (Microsoft Learn, WSL)",
      "kind": "official-documentation",
      "url": "https://learn.microsoft.com/en-us/windows/wsl/wsl-config",
      "title": "Advanced settings configuration in WSL",
      "version_or_date": "current docs page at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: the wsl-config page (full page text)"],
      "excerpt": "In the [wsl2] table: 'vmIdleTimeout(1) | number | 60000 | The number of milliseconds that a VM is idle, before it is shut down.' Footnote 1 states 'Only available on Windows 11.' The page defines .wslconfig as global WSL 2 settings in %UserProfile%\\.wslconfig. Full-text search of the fetched page found no occurrence of 'child of', 'init process', 'systemd', 'idle' beyond that row, and no documented value of -1 for vmIdleTimeout.",
      "evidence_limits": "The official page defines neither what 'idle' means in terms of processes nor the -1 sentinel."
    },
    {
      "id": "s9",
      "publisher": "Microsoft (Windows Command Line blog, devblogs.microsoft.com)",
      "kind": "publisher-blog",
      "url": "https://devblogs.microsoft.com/commandline/systemd-support-is-now-available-in-wsl/",
      "title": "Systemd support is now available in WSL!",
      "version_or_date": "published 2022-09-21",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: the systemd-support blog post"],
      "excerpt": "'It is also important to note that with these change, systemd services will NOT keep your WSL instance alive. Your WSL instance will stay alive in the same way it did before, which you can read more about here.' The 'here' link points to https://devblogs.microsoft.com/commandline/background-task-support-in-wsl/.",
      "evidence_limits": "A blog post, not reference documentation; it states that systemd services do not keep the instance alive but does not state the child-of-init rule itself."
    },
    {
      "id": "s10",
      "publisher": "Microsoft WSL engineering (GitHub issue comment, microsoft/WSL)",
      "kind": "issue-tracker-comment",
      "url": "https://github.com/microsoft/WSL/issues/8854",
      "title": "Distro shut down even with running processes",
      "version_or_date": "issue and comments present at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "search-snippet-only",
      "tool_calls": ["web_search auto: Microsoft WSL documentation distribution terminates when no processes remain init process"],
      "excerpt": "Snippet only: 'Unfortunately, this is the new documented behavior, per blog post: It is also important to note that with these change, systemd services will NOT keep your WSL instance alive. You've currently got to have something running as a child of _init_ - the Microsoft init, not pid 1 - to keep the distro running.'",
      "evidence_limits": "NOT FETCHED. Search snippets are not evidence; recorded only to explain where the child-of-init phrasing circulates."
    },
    {
      "id": "s11",
      "publisher": "Microsoft WSL maintainers (GitHub issue comment, microsoft/WSL)",
      "kind": "issue-tracker-comment",
      "url": "https://github.com/microsoft/WSL/issues/9667",
      "title": "WSL terminates systemd service docker containers when last terminal is closed",
      "version_or_date": "issue and comments present at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "search-snippet-only",
      "tool_calls": ["web_search auto: WSL2 VM shuts down when no processes running child of init documentation vmIdleTimeout idle definition"],
      "excerpt": "Snippet only: 'This is by design. The VM hosting WSL will idle-terminate when: 1. There are no open terminal windows and 2. No backgroud processes were launched by the user explicitly (things launched by systemd don't count)' and 'We do have an undocumented config option that disables this behavior'.",
      "evidence_limits": "NOT FETCHED. Snippet only; the speaker identity and current status were not verified on the page."
    },
    {
      "id": "s12",
      "publisher": "Microsoft WSL maintainers (GitHub issue comment, microsoft/WSL)",
      "kind": "issue-tracker-comment",
      "url": "https://github.com/microsoft/WSL/issues/10138",
      "title": "How to make wsl2 alive in the background",
      "version_or_date": "issue and comments present at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "search-snippet-only",
      "tool_calls": ["web_search auto: WSL2 VM shuts down when no processes running child of init documentation vmIdleTimeout idle definition"],
      "excerpt": "Snippet only: 'You can force WSL2 to keep running with: [wsl2] vmIdleTimeout=-1 in %userprofile%/.wslconfig' and 'I'm not sure why this is not documented at https://learn.microsoft.com/en-us/windows/wsl/wsl-config'.",
      "evidence_limits": "NOT FETCHED. Snippet only; cannot be used as validated evidence."
    },
    {
      "id": "s13",
      "publisher": "The Go project (pkg.go.dev, standard library documentation)",
      "kind": "official-documentation",
      "url": "https://pkg.go.dev/crypto/x509",
      "title": "x509 package - crypto/x509",
      "version_or_date": "package docs rendered for go1.27.1 at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": [
        "fetch_content read: https://pkg.go.dev/crypto/x509",
        "web_search auto: Go crypto/x509 macOS system keychain roots CGO_ENABLED=0 verification documentation"
      ],
      "excerpt": "Package overview: 'On macOS and Windows, certificate verification is handled by system APIs, but the package aims to apply consistent validation rules across operating systems.' Certificate.Verify: 'If opts.Roots is nil, the platform verifier might be used, and verification details might differ from what is described below. If system roots are unavailable the returned error will be of type SystemRootsError.' SystemCertPool: 'On platforms which have system APIs for certificate verification (macOS and Windows), setting SSL_CERT_FILE or SSL_CERT_DIR will prevent those APIs from being used, unless the x509sslcertoverrideplatform=0 GODEBUG setting is used. (This changed in Go 1.27.)'",
      "evidence_limits": "The user-facing docs never mention cgo, CGO_ENABLED, keychains, or the security(1) CLI."
    },
    {
      "id": "s14",
      "publisher": "The Go project (pkg.go.dev)",
      "kind": "official-documentation",
      "url": "https://pkg.go.dev/crypto/x509/internal/macos",
      "title": "macos package - crypto/x509/internal/macos",
      "version_or_date": "rendered for darwin/amd64, go1.26.5 series at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: https://pkg.go.dev/crypto/x509/internal/macos"],
      "excerpt": "'Package macos provides cgo-less wrappers for Core Foundation and Security.framework, similarly to how package syscall provides access to libSystem.dylib.' Exported helpers include SecCertificateCopyData, 'func SecTrustEvaluateWithError(trustObj CFRef) (int, error)', and SecTrustSetVerifyDate.",
      "evidence_limits": "Internal package doc; it documents the cgo-less claim but not keychain trust semantics."
    },
    {
      "id": "s15",
      "publisher": "The Go project (go.dev source browser)",
      "kind": "publisher-source",
      "url": "https://go.dev/src/crypto/x509/root_darwin.go?m=text",
      "title": "src/crypto/x509/root_darwin.go",
      "version_or_date": "master at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: https://go.dev/src/crypto/x509/root_darwin.go?m=text", "web_search auto: golang x509 macOS SecTrustEvaluate cgo required keychain trusted certificate not used without cgo"],
      "excerpt": "The file has no cgo import; it imports 'crypto/x509/internal/macos'. systemVerify: builds a mutable CFArray, creates the leaf via macos.SecCertificateCreateWithData, appends intermediates, creates 'sslPolicy, err := macos.SecPolicyCreateSSL(opts.DNSName)', then 'trustObj, err := macos.SecTrustCreateWithCertificates(certs, policies)', optionally SecTrustSetVerifyDate, then 'if ret, err := macos.SecTrustEvaluateWithError(trustObj); err != nil' mapping ErrSecCertificateExpired, ErrSecHostNameMismatch and ErrSecNotTrusted to Go error types, and copies the resulting chain via macos.SecTrustCopyCertificateChain. Comment in that path: 'The linker will not include these unused functions in binaries built with cgo enabled' appears in the historical fallback file, not this one.",
      "evidence_limits": "Source-level evidence, not prose documentation."
    },
    {
      "id": "s16",
      "publisher": "Microsoft (Windows Command Line blog)",
      "kind": "publisher-blog",
      "url": "https://devblogs.microsoft.com/commandline/background-task-support-in-wsl/",
      "title": "Background Task Support in WSL",
      "version_or_date": "older blog post referenced by s9",
      "retrieved_at": "2026-09-14",
      "retrieval": "fetched",
      "tool_calls": ["fetch_content read: the background-task-support blog post"],
      "excerpt": "'Starting in Windows Insiders Build 17046, WSL supports background tasks (including daemons)... these processes will continue running in the background even after the last console window has been closed.' The post discusses tmux and startup tasks and does NOT state any child-of-init keep-alive rule or vmIdleTimeout semantics.",
      "evidence_limits": "Negative result: the page linked from s9 as the definition of keep-alive behaviour does not contain the child-of-init rule."
    },
    {
      "id": "s17",
      "publisher": "Microsoft (microsoft/dev-tunnels issue tracker)",
      "kind": "issue-tracker",
      "url": "https://github.com/microsoft/dev-tunnels/issues/511",
      "title": "CLI managed identity authentication does not work with non-IMDS token endpoints or system-assigned managed identities",
      "version_or_date": "issue present at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "search-snippet-only",
      "tool_calls": ["web_search auto: Dev Tunnels headless authentication access token service principal without browser"],
      "excerpt": "Snippet only: 'devtunnel user login allows managed identity authentication using --mi-client-id, --mi-object-id or --mi-resource-id. The implementation in Microsoft.DevTunnels.Cli.Authentication.ManagedIdentityApp.AcquireTokenAsync tries to acquire an access token from the IMDS managed identity token endpoint'.",
      "evidence_limits": "NOT FETCHED. Snippet only; cannot be used as validated evidence and is not documented on Microsoft Learn."
    },
    {
      "id": "s18",
      "publisher": "Microsoft (microsoft/dev-tunnels issue tracker)",
      "kind": "issue-tracker",
      "url": "https://github.com/microsoft/dev-tunnels/issues/443",
      "title": "AADSTS100007 Only managed identities and Microsoft internal service identities are supported. SN+I authentication is required.",
      "version_or_date": "issue present at retrieval time",
      "retrieved_at": "2026-09-14",
      "retrieval": "search-snippet-only",
      "tool_calls": ["web_search auto: Dev Tunnels headless authentication access token service principal without browser"],
      "excerpt": "Snippet only: 'we are using devtunnel and login with service principal, It worked very well before, but I encountered this problem in the past two days... AADSTS100007: This request was received by an Azure AD regional authentication endpoint.'",
      "evidence_limits": "NOT FETCHED. Snippet only; indicates service-principal login exists but is fragile, and cannot be used as validated evidence."
    }
  ],
  "claims": [
    {
      "id": "c1",
      "question": "Q1",
      "statement": "Issue cloudflare/cloudflared#1673 exists, is OPEN, was created 2026-06-10T18:09:37Z, has zero comments, no labels, no assignees and no closure.",
      "status": "verified",
      "sources": ["s1"],
      "basis": "GitHub API JSON and rendered issue page both read directly."
    },
    {
      "id": "c2",
      "question": "Q1",
      "statement": "The behavioural report in #1673 is that cloudflared 2026.6.0 ignores service-token flags, TUNNEL_SERVICE_TOKEN_ID/SECRET env vars and -H header injection for 'access tcp' and 'access ssh', falling through to interactive browser auth per TCP connection, with the SSH banner never delivered, and that this is a regression from 2026.5.1.",
      "status": "verified-as-report-only",
      "sources": ["s1"],
      "basis": "Verbatim issue body fetched. Single reporter (author_association NONE); no maintainer confirmation, no second reproduction, no linked PR, no release note."
    },
    {
      "id": "c3",
      "question": "Q1",
      "statement": "The generalization 'versions at or above 2026.6.0 ignore service tokens' is NOT established. Only one version (2026.6.0) is named by the reporter, and no source states whether 2026.6.1, 2026.7.x, 2026.8.x or later behave the same way or are fixed.",
      "status": "unverified",
      "sources": ["s1", "s3"],
      "basis": "The issue title and body name 2026.6.0 specifically. The changelog (s3) contains no service-token or access-tcp entry from 2026.6.0 onward."
    },
    {
      "id": "c4",
      "question": "Q1",
      "statement": "No publisher evidence shows that any release after 2026.6.0 restores headless service-token authentication for 'access ssh' or 'access tcp'.",
      "status": "negative-documentation-finding",
      "sources": ["s1", "s3"],
      "basis": "The issue remains open with zero comments, and full-text search of RELEASE_NOTES for 'service token' and 'access tcp' returned no matches; the latest access-ssh changelog line is 2025-03-17."
    },
    {
      "id": "c5",
      "question": "Q1",
      "statement": "Release 2026.5.1 exists as a published GitHub release and publishes SHA256 checksums for its artifacts, including cloudflared-linux-amd64 3c6a5ba995a258dbe90f98e5fdb2c2620b7be72c3ca761614f6eb52aee252cea and cloudflared-darwin-arm64.tgz 086dd6628a3608eba7327af1092abcc52b10994524112ae49bf356ec49bef8c4.",
      "status": "verified",
      "sources": ["s2", "s3"],
      "basis": "Checksum block fetched from the release page; the changelog lists 2026.5.1 entries dated 2026-05-22."
    },
    {
      "id": "c6",
      "question": "Q2",
      "statement": "cloudflared's tunnel protocol parameter defaults to 'auto', is controlled by TUNNEL_TRANSPORT_PROTOCOL, accepts auto/http2/quic, and under 'auto' configures QUIC and falls back to http2 when UDP connections cannot be established.",
      "status": "verified",
      "sources": ["s4"],
      "basis": "Cloudflare's own run-parameters reference text fetched verbatim."
    },
    {
      "id": "c7",
      "question": "Q2",
      "statement": "quic runs over UDP and http2 runs over TCP on port 7844 to Cloudflare's global network.",
      "status": "verified",
      "sources": ["s5"],
      "basis": "Cloudflare firewall reference: port 7844 'via UDP if using the quic protocol or TCP if using the http2 protocol'; per-SNI rows show quic.cftunnel.com UDP and h2.cftunnel.com TCP."
    },
    {
      "id": "c8",
      "question": "Q2",
      "statement": "Setting the protocol parameter (or TUNNEL_TRANSPORT_PROTOCOL) to 'http2' selects HTTP/2 rather than the automatic QUIC-plus-fallback behaviour, because 'http2' is a documented explicit value of the same parameter whose only documented automatic/fallback behaviour belongs to 'auto'.",
      "status": "verified",
      "sources": ["s4"],
      "basis": "Parameter semantics as documented. Note: the docs do not use the word 'force'; the wording above is the strongest statement the source supports."
    },
    {
      "id": "c9",
      "question": "Q2",
      "statement": "Cloudflare states that post-quantum key agreements are not supported when using the http2 protocol, while QUIC tunnel connections are post-quantum encrypted by default with fallback to non-PQ unless --post-quantum is given.",
      "status": "verified",
      "sources": ["s4"],
      "basis": "Verbatim run-parameters text. Consequence for slice 1: forcing http2 as a transport workaround forfeits PQ key agreement on the tunnel transport."
    },
    {
      "id": "c10",
      "question": "Q2",
      "statement": "The specific hybrid group name (X25519MLKEM768) claimed for QUIC/H2 edge connections was seen only in a commit-message search snippet and is not validated here.",
      "status": "unverified",
      "sources": [],
      "basis": "Commit page was not fetched; no fetched source names the hybrid group."
    },
    {
      "id": "c11",
      "question": "Q3",
      "statement": "'devtunnel connect TUNNELID' forwards a client-local listening port to a dev tunnel port (both IPv4 and IPv6), and Microsoft states the dev tunnel service builds on the standard SSH protocol for port-forwarding.",
      "status": "verified",
      "sources": ["s6"],
      "basis": "Learn CLI reference: 'SSH: Forwarding from 127.0.0.1:3000 to host port 3000.' plus the explicit parenthetical about the SSH prefix."
    },
    {
      "id": "c12",
      "question": "Q3",
      "statement": "An SSH client can use that forwarded local TCP port: the client-side endpoint is a plain local port that any local application can dial.",
      "status": "verified-as-report-only",
      "sources": ["s6"],
      "basis": "The docs describe the local endpoint generically and their examples are HTTP/web; no Microsoft page shows an SSH client connecting through a dev tunnel, so this remains inference from the documented behaviour rather than a documented scenario."
    },
    {
      "id": "c13",
      "question": "Q3",
      "statement": "Dev Tunnels does not document a raw-TCP port protocol: 'devtunnel port create --protocol' documents only http, https and auto.",
      "status": "negative-documentation-finding",
      "sources": ["s6"],
      "basis": "Fetched CLI reference text; no 'tcp' value appears in the port-protocol options."
    },
    {
      "id": "c14",
      "question": "Q3",
      "statement": "Headless (no browser) authentication is documented: 'devtunnel token TUNNELID --scopes connect' issues a client access token, most CLI commands accept --access-token instead of logging in, web clients pass 'X-Tunnel-Authorization: tunnel <TOKEN>', tokens expire in about 24 hours, and Microsoft states this is useful for non-interactive clients.",
      "status": "verified",
      "sources": ["s6", "s7"],
      "basis": "Learn security page Tip plus the CLI reference's token command."
    },
    {
      "id": "c15",
      "question": "Q3",
      "statement": "Service-principal authentication for Dev Tunnels is not documented on Microsoft Learn; the documented non-interactive-but-human path is device-code login (devtunnel user login -d), and managed-identity flags appear only in unfetched GitHub issue snippets.",
      "status": "negative-documentation-finding",
      "sources": ["s6", "s7"],
      "basis": "Full-text of the fetched CLI reference lists only 'devtunnel user login' with -g and -d; the security page never says 'service principal'. s17/s18 are snippet-only and unusable as evidence."
    },
    {
      "id": "c16",
      "question": "Q3",
      "statement": "Dev Tunnels requires authentication by default for both hosting and connecting and does not support anonymous hosting; anonymous access requires an explicit allow-anonymous access control entry.",
      "status": "verified",
      "sources": ["s6", "s7"],
      "basis": "Learned from both fetched Microsoft pages."
    },
    {
      "id": "c17",
      "question": "Q4",
      "statement": "The Go project documents that on macOS and Windows certificate verification is handled by system APIs, and that Certificate.Verify uses the platform verifier when opts.Roots is nil.",
      "status": "verified",
      "sources": ["s13"],
      "basis": "Verbatim pkg.go.dev text."
    },
    {
      "id": "c18",
      "question": "Q4",
      "statement": "The macOS platform verifier is implemented without cgo: crypto/x509/internal/macos is documented as 'cgo-less wrappers for Core Foundation and Security.framework', and root_darwin.go's systemVerify creates a Security.framework trust object with an SSL policy and calls SecTrustEvaluateWithError with no cgo import.",
      "status": "verified",
      "sources": ["s14", "s15"],
      "basis": "Package documentation plus fetched standard-library source."
    },
    {
      "id": "c19",
      "question": "Q4",
      "statement": "A Go program should leave opts.Roots nil and must not set SSL_CERT_FILE or SSL_CERT_DIR, because on macOS those variables prevent the system verification APIs from being used unless GODEBUG=x509sslcertoverrideplatform=0 (behaviour changed in Go 1.27).",
      "status": "verified",
      "sources": ["s13"],
      "basis": "Verbatim SystemCertPool and Verify documentation; this is the documented condition under which platform/keychain trust is consulted."
    },
    {
      "id": "c20",
      "question": "Q4",
      "statement": "Because systemVerify delegates to SecTrustCreateWithCertificates plus SecTrustEvaluateWithError, certificate trust decisions (including trust configured through macOS keychains) are made by the system rather than by Go's own root list.",
      "status": "verified-as-report-only",
      "sources": ["s15", "s14"],
      "basis": "Source-level evidence: the fetched code calls the system trust evaluator. Go's user-facing documentation does not describe keychain semantics, so the keychain consequence is reported at source level only."
    },
    {
      "id": "c21",
      "question": "Q4",
      "statement": "Nothing in Go's publisher documentation requires shelling out to security(1), and the current darwin verification path contains no exec of the security CLI.",
      "status": "negative-documentation-finding",
      "sources": ["s13", "s14", "s15"],
      "basis": "Full-text of the fetched x509 docs never mentions security(1); the fetched source path uses only the internal/macos wrappers."
    },
    {
      "id": "c22",
      "question": "Q5",
      "statement": "Microsoft documents vmIdleTimeout in the .wslconfig [wsl2] section as default 60000, 'The number of milliseconds that a VM is idle, before it is shut down', available only on Windows 11.",
      "status": "verified",
      "sources": ["s8"],
      "basis": "Verbatim table row and footnote from the fetched Microsoft Learn page."
    },
    {
      "id": "c23",
      "question": "Q5",
      "statement": "Microsoft's .wslconfig reference does not define what 'idle' means, does not mention processes or children of init in that row, and does not document vmIdleTimeout=-1.",
      "status": "negative-documentation-finding",
      "sources": ["s8"],
      "basis": "Full page fetched and searched; only the single row sentence exists, and the period table has no -1 entry."
    },
    {
      "id": "c24",
      "question": "Q5",
      "statement": "Microsoft statements outside the .wslconfig reference say the WSL instance/VM is kept alive by user-launched processes and not by systemd services, with the keep-alive condition described in issue threads as needing something running as a child of the Microsoft init.",
      "status": "verified-as-report-only",
      "sources": ["s9", "s16", "s10", "s11", "s12"],
      "basis": "The Microsoft devblog statement is fetched and verbatim; the child-of-init phrasing and the vmIdleTimeout=-1 sentinel appear only in GitHub issue comments that were seen as search snippets and were NOT fetched, so they carry report-only weight."
    },
    {
      "id": "c25",
      "question": "Q5",
      "statement": "The specific formulation 'Microsoft documents that the distribution shuts down when no process that is a child of the init process remains' is unsupported: no fetched Microsoft documentation page contains that rule, and the blog page linked as the authoritative explanation of keep-alive does not contain it either.",
      "status": "negative-documentation-finding",
      "sources": ["s8", "s9", "s16"],
      "basis": "The .wslconfig page defines only millisecond idle time; the linked background-tasks post limits itself to console windows and background tasks."
    }
  ],
  "question_summary": [
    {"question": "Q1", "topic": "cloudflared service tokens and the 2026.5.1 pin", "status": "partial", "detail": "Issue state, single-version scope, absence of a documented fix and the 2026.5.1 checksums are settled; the '>= 2026.6.0' generalization and headless service-token behaviour on later releases remain unverified."},
    {"question": "Q2", "topic": "tunnel transport protocol and PQ consequence", "status": "done", "detail": "All parts answered from Cloudflare documentation and verified."},
    {"question": "Q3", "topic": "Dev Tunnels raw TCP and headless auth", "status": "partial", "detail": "Local port forwarding and token-based headless auth are documented; an SSH-client scenario and any service-principal path are not documented."},
    {"question": "Q4", "topic": "macOS trust store from Go", "status": "done", "detail": "Platform verification via Security.framework without cgo is documented and confirmed in source; no documented need to shell out to security(1)."},
    {"question": "Q5", "topic": "WSL2 lifetime and vmIdleTimeout", "status": "partial", "detail": "The documented millisecond semantics are verified; the child-of-init shutdown rule exists only as report-level Microsoft statements, not in reference documentation."}
  ],
  "carried_forward_unverified": [
    "Whether cloudflared 2026.6.1, 2026.7.x, 2026.8.x or later ignore service tokens for access ssh/access tcp the way 2026.6.0 is reported to (reproduce before pinning or unpinning).",
    "Whether the 2026.6.0 service-token failure is reproducible at all on non-macOS platforms and with a plain flags-only invocation (the report is single-source, macOS arm64).",
    "Whether the failure, if real, also affects 'cloudflared access curl' or header-injected HTTP paths (the reporter says HTTP curl with service-token headers works).",
    "Whether an SSH client can actually complete banner exchange through 'devtunnel connect' (no publisher example; requires a spike).",
    "Whether Dev Tunnels service-principal or managed-identity login is supported and stable (only GitHub issue snippets, not Microsoft documentation).",
    "Whether Dev Tunnels exposes a raw-TCP port protocol (documented port protocols are http/https/auto only).",
    "The post-quantum hybrid group name used on QUIC/H2 edge connections.",
    "The documented definition of WSL 'idle' and the vmIdleTimeout=-1 sentinel (neither appears in the Microsoft .wslconfig reference)."
  ],
  "product_choices_excluded": [
    "Which cloudflared version to pin, whether to force http2, whether to keep or drop the own-domain requirement, how the macOS trust probe is implemented, and what the WSL2 keepalive must do are product decisions owned by the orchestrator; this artifact records only evidence and its status."
  ],
  "notes_for_orchestrator": [
    "Outcome is partial by design: question 2 and question 4 are fully source-backed, while questions 1, 3 and 5 carry report-only or negative-documentation findings, so proposal_ready stays false.",
    "Because the open-web class is recorded partial (degraded search coverage and inconclusive source_check verdicts), the parent should pass working provider credentials or a narrower source list next run if stronger search corroboration is required.",
    "Two slice-1 decisions referenced in the retained intent rely on partial findings: the macOS trust-store probe is safe to specify (Q4 done), while the verdict wording about QUIC failing and TCP succeeding should avoid asserting that a forced http2 path is equivalent, because Cloudflare documents that http2 loses post-quantum key agreement.",
    "No digest for revision 2 could be computed: this child had no execution tool to hash the document bytes, so the locator digest is recorded as null rather than guessed."
  ]
}