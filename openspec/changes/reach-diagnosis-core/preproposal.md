{
  "schema": "gentle-ai.sdd-preproposal/v1",
  "revision": 3,
  "change": "reach-diagnosis-core",
  "worktree": "/home/luisalt20/projects/close/herdr-reach",
  "artifact_store": "both",
  "exploration_reference": {
    "openspec": "openspec/changes/reach-diagnosis-core/explore.md",
    "revision": 1,
    "read_status": "not-read",
    "read_status_reason": "The bounded artifact scope carries only the research and preproposal locators; any read of explore.md is refused by the scope guard, so this phase used the inlined exploration summary supplied by the parent and never opened the file."
  },
  "research_request": {
    "origin": "retained_intent supplied by the parent",
    "retained_intent": "R1a headless diagnosis core: verify the PRD external claims (cloudflared pin and service tokens, QUIC/http2, Dev Tunnels, macOS trust store, WSL2 vmIdleTimeout) before the slice-1 proposal depends on them.",
    "questions": [
      {
        "id": "Q1",
        "text": "cloudflared service tokens: confirm or refute that versions at or above 2026.6.0 ignore service tokens for cloudflared access ssh and access tcp and fall through to an interactive browser flow, that issue 1673 documents it, the exact affected versions, whether any later release honors service tokens headlessly, and whether release 2026.5.1 exists with a published checksum."
      },
      {
        "id": "Q2",
        "text": "Tunnel transport protocol: confirm that cloudflared prefers QUIC and falls back to HTTP/2 over TCP, that TUNNEL_TRANSPORT_PROTOCOL=http2 forces HTTP/2, and what the publisher states about the post-quantum key agreement consequence of forcing HTTP/2 on the tunnel transport."
      },
      {
        "id": "Q3",
        "text": "Microsoft Dev Tunnels: can it forward raw TCP so an SSH client can use it, and can it authenticate headlessly with a token or a service principal rather than a browser?"
      },
      {
        "id": "Q4",
        "text": "macOS trust store from Go: can a Go program verify a certificate against system and keychain-trusted roots without cgo and without shelling out to security? Publisher documentation only."
      },
      {
        "id": "Q5",
        "text": "WSL2 lifetime: what vmIdleTimeout actually controls per Microsoft documentation, and whether Microsoft documents that the distribution shuts down when no process that is a child of the init process remains."
      }
    ],
    "source_classes_requested": [
      "documentation",
      "open-web"
    ],
    "source_restrictions": [
      "Question 4 requires publisher documentation only.",
      "Generic MCP and dynamic namespace gateways are not approved evidence routes."
    ],
    "immutability": "The request above is carried intent and was treated as immutable; no question was added, removed or reworded."
  },
  "classes_selected": [
    {
      "class": "documentation",
      "selected": true,
      "declared_tools": [
        "fetch_content"
      ],
      "observed_active_tools": [
        "fetch_content"
      ],
      "outcome": "available",
      "tool_calls_made": 9,
      "evidence_used": [
        "s2",
        "s4",
        "s5",
        "s6",
        "s7",
        "s8",
        "s9",
        "s13",
        "s14",
        "s15",
        "s16"
      ]
    },
    {
      "class": "open-web",
      "selected": true,
      "declared_required_tools": [
        "web_search",
        "source_check",
        "fetch_content",
        "get_search_content"
      ],
      "observed_active_tools": [
        "web_search",
        "source_check",
        "fetch_content",
        "get_search_content"
      ],
      "outcome": "partial",
      "detail": "All four required tools executed. web_search lost one batch to an Exa 429 rate limit and explicit brave/openai providers lacked keys; three source_check calls returned inconclusive 'unclear' verdicts and were not used as evidence. Fetched originals supplied all validated claims.",
      "evidence_used": [
        "s1",
        "s3"
      ]
    }
  ],
  "admission": {
    "outcome": "partial",
    "reason": "Both selected classes ran and both are recorded, but the open-web class is partial, no selection extensions map was carried, and several questions resolved to report-only or negative-documentation findings rather than validated answers.",
    "extension_provenance": "None carried; no --extension paths asserted.",
    "denied_routes": [
      "generic MCP and dynamic gateway routes"
    ],
    "denied_class_reasons": []
  },
  "evidence_references": [
    {
      "artifact": "research",
      "store": "openspec",
      "path": "openspec/changes/reach-diagnosis-core/research.md",
      "revision": 2
    },
    {
      "artifact": "research",
      "store": "engram",
      "topic_key": "sdd/reach-diagnosis-core/research",
      "revision": 2
    },
    {
      "source_ids": [
        "s1",
        "s2",
        "s3",
        "s4",
        "s5",
        "s6",
        "s7",
        "s8",
        "s9",
        "s10",
        "s11",
        "s12",
        "s13",
        "s14",
        "s15",
        "s16",
        "s17",
        "s18"
      ]
    }
  ],
  "research_result": {
    "schema": "gentle-ai.sdd-research/v1",
    "revision": 2,
    "outcome": "partial",
    "question_status": {
      "Q1": "partial",
      "Q2": "done",
      "Q3": "partial",
      "Q4": "done",
      "Q5": "partial"
    },
    "validated_highlights": [
      "Cloudflare documents that the tunnel protocol parameter defaults to auto, prefers quic and falls back to http2 when UDP is unavailable, and that post-quantum key agreements are not supported when using http2 (c6, c7, c9).",
      "Release 2026.5.1 exists with published SHA256 checksums (c5).",
      "Dev Tunnels forwards a local client port to a tunnel port and supports token-based non-interactive authentication (c11, c14).",
      "Go verifies macOS certificates through cgo-less Security.framework wrappers and documents the platform verifier path (c17, c18, c19).",
      "Microsoft documents vmIdleTimeout as 'number of milliseconds that a VM is idle, before it is shut down', default 60000, Windows 11 only (c22)."
    ],
    "unresolved_items": [
      "Whether releases after 2026.6.0 ignore service tokens for access ssh/access tcp (c3, c4).",
      "Whether Dev Tunnels can carry an SSH client end-to-end and whether service-principal login is supported (c12, c15).",
      "Whether Microsoft documentation states the child-of-init shutdown rule or the vmIdleTimeout=-1 sentinel (c23, c25)."
    ]
  },
  "product_decisions": "confirmed",
  "proposal_ready": true,
  "proposal_ready_reason": "Research outcome is partial: Q2 and Q4 are fully source-backed, but Q1, Q3 and Q5 carry report-only or negative-documentation findings, the open-web class is recorded partial, and product decisions remain unconfirmed by the parent. Under the research contract any selected partial class keeps proposal_ready false.",
  "artifact_locators": {
    "carried_revision_1": [
      {
        "artifact": "research",
        "digest": "e7472061a2f3ab6e26a9e94688a4b2c4d84a83909a1b64dfc5de9b14b9a7e16c",
        "engram": {
          "id": 793,
          "project": "herdr-reach",
          "topic_key": "sdd/reach-diagnosis-core/research",
          "revision_count": 1
        },
        "path": "openspec/changes/reach-diagnosis-core/research.md",
        "revision": 1
      },
      {
        "artifact": "preproposal",
        "digest": "4c4006b3e9b3c2f44bcfef1ce4143db616d8d646c64ccb0d497b79609348cb38",
        "engram": {
          "id": 792,
          "project": "herdr-reach",
          "topic_key": "sdd/reach-diagnosis-core/preproposal",
          "revision_count": 1
        },
        "path": "openspec/changes/reach-diagnosis-core/preproposal.md",
        "revision": 1
      }
    ],
    "written_revision_2": [
      {
        "artifact": "research",
        "digest": null,
        "digest_status": "not-computed-no-exec-tool-in-this-child",
        "path": "openspec/changes/reach-diagnosis-core/research.md",
        "revision": 2,
        "topic_key": "sdd/reach-diagnosis-core/research",
        "project": "herdr-reach",
        "store": "both",
        "byte_identical_across_stores": true
      },
      {
        "artifact": "preproposal",
        "digest": null,
        "digest_status": "not-computed-no-exec-tool-in-this-child",
        "path": "openspec/changes/reach-diagnosis-core/preproposal.md",
        "revision": 2,
        "topic_key": "sdd/reach-diagnosis-core/preproposal",
        "project": "herdr-reach",
        "store": "both",
        "byte_identical_across_stores": true
      }
    ]
  },
  "persistence": {
    "openspec": [
      {
        "path": "openspec/changes/reach-diagnosis-core/research.md",
        "revision": 2,
        "operation": "full-write"
      },
      {
        "path": "openspec/changes/reach-diagnosis-core/preproposal.md",
        "revision": 2,
        "operation": "full-write"
      }
    ],
    "engram": [
      {
        "topic_key": "sdd/reach-diagnosis-core/research",
        "project": "herdr-reach",
        "revision": 2,
        "operation": "full-content-save"
      },
      {
        "topic_key": "sdd/reach-diagnosis-core/preproposal",
        "project": "herdr-reach",
        "revision": 2,
        "operation": "full-content-save"
      }
    ],
    "mutation_scope": "Only the two carried artifact paths and their matching Engram topics were mutated. No Go sources, explore.md, PRD.md or README.md were touched; nothing was staged or committed.",
    "readback_note": "Revision 1 of both artifacts was read back from both stores before any write. Revision 2 was freshly read back after writing to confirm identity; a save acknowledgement alone was not treated as durability."
  },
  "next_steps_for_orchestrator": [
    "Confirm product decisions (pin policy, http2 trade-off wording, own-domain requirement, WSL2 keepalive expectations) before admitting a proposal, because those choices are not settled by this artifact.",
    "If the slice-1 verdict wording or the pinned-version decision must be fully evidence-backed, re-run research with working search provider credentials or with an SSH-through-dev-tunnel spike to convert the reported items into validated findings.",
    "Treat the four carried-forward unverified items about cloudflared service tokens as explicit unknowns rather than as facts in the proposal."
  ],
  "orchestrator_confirmation": {
    "confirmed_at": "2026-09-14",
    "confirmed_by": "user (explicit answers in session)",
    "decisions": [
      "Artifact acceptance: research stays recorded as outcome=partial with proposal_ready=false in the research artifact. The verified findings (Q2 transport/PQ, Q4 macOS trust store, release checksums and issue state) are citable for this change; the unverified items are carried as explicit risks. The user accepted this handoff explicitly rather than attempting another collection run.",
      "Slice scope: the R1 split is pre-decided. R1a is this change (Go module bootstrap, probe suite, diagnosis engine, transport feasibility as detect-and-recommend, headless --json report, zero system mutation). R1b (TUI shell, router, role/diagnose/verdict screens) is a separate later change.",
      "macOS trust-store probe: report indeterminate on macOS with the documented limitation (Go cannot enumerate macOS system roots; keychain trust is only visible through the platform verifier with nil Roots). Chosen to keep R1a small and offline-testable, with the limitation stated in the verdict output.",
      "Program roadmap R1-R12 approved as proposed by the explore artifact."
    ],
    "note": "proposal_ready=true below states that this handoff is confirmed for the proposal phase under the acceptance decision above. It is NOT a claim that research completed: the research artifact keeps outcome=partial and proposal_ready=false."
  },
  "proposal_ready_basis": "Orchestrator-confirmed handoff: research recorded as partial and explicitly accepted by the user for R1a; verified findings citable; unverified items carried as risks in the research artifact."
}