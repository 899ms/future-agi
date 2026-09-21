# Claude Agent SDK → AgentCC → Vertex Gemini

This branch contains only the production gateway path: Anthropic-compatible requests from
Claude Agent SDK can resolve a Claude-shaped model alias to Vertex Gemini; AgentCC can hold
a pasted Vertex service-account JSON in encrypted platform credentials and refresh its OAuth
tokens. It does not change ALK's self-repair loop or switch any running job to production.

## Production configuration

1. Deploy the platform API/frontend and AgentCC gateway from this branch together. The
   generated admin contract carries `service_account_json` between them.
2. In AgentCC **Add Provider**, choose **Google Vertex AI**. Enter the Google Cloud project,
   Vertex location, paste the service-account JSON, and register the model as
   `vertex_ai/gemini-3.7-flash`. The JSON is encrypted by the platform and not returned to
   the browser after saving. The service account needs permission to invoke Vertex AI models
   in the selected project.
3. In the production gateway's **static routing configuration**, give the access group assigned
   to the harness virtual key this alias. The existing access-group checker reads gateway
   configuration, not the per-organization UI routing form:

   ```yaml
   routing:
     access_groups:
       alk_authoring:
         models:
           - vertex_ai/gemini-3.7-flash
         aliases:
           claude-sonnet-4-6: vertex_ai/gemini-3.7-flash
   ```

   Merge this into the existing routing configuration; do not replace other groups or rules.
   Create a production virtual key for the same organization as the Vertex provider and assign
   it the `alk_authoring` access group. Keep the key in a secret manager.
4. After deployment, point a **local** ALK test at the production gateway:

   ```bash
   export ALK_HARNESS=claude
   export ALK_HARNESS_MODEL=vertex_ai/gemini-3.7-flash
   export AGENTCC_BASE_URL=https://gateway.futureagi.com
   export AGENTCC_API_KEY=<production-virtual-key-from-secret-manager>
   ```

   ALK sends `claude-sonnet-4-6` on the Anthropic wire because Claude Code validates model
   names locally. AgentCC resolves that alias to `vertex_ai/gemini-3.7-flash`, routes to the
   organization's Vertex provider, and returns the Claude-shaped model name to the SDK.

Do not reuse `AGENTCC_INTERNAL_API_KEY` as the production virtual key. For hosted ALK jobs,
the separate platform-side `AGENTCC_HARNESS_API_KEY`/`AGENTCC_BASE_URL` configuration belongs
to the self-repair branch and is not part of this production gateway deployment.

## Verification gate

Before moving a full harness run, verify that `/v1/messages` accepts the virtual key and
Claude alias, produces a response from the intended Gemini provider, and completes a short
streaming tool-use exchange. A 401 means key/auth scope is wrong; a model-not-available error
usually means the key's access group, alias, or provider model registration is missing.
