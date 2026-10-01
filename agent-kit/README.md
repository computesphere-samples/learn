# Agent kit: deploy to ComputeSphere with a coding agent

Coding agents can already deploy: give one the `csph` CLI or ComputeSphere's MCP server and it will. What they don't bring is the judgment: a token that reaches one project, a health check that means something, secrets kept out of the code, and a human's yes before anything live changes. This kit writes that judgment down so your agent starts every session with it.

| File | What it is |
| --- | --- |
| [`AGENTS.md`](AGENTS.md) | Drop-in instructions for any repository: how to deploy to ComputeSphere, the rules, and a troubleshooting order |
| [`SKILL.md`](SKILL.md) | The same guidance as a step-by-step skill, for agents that load skills on demand |

## Use `AGENTS.md`

Copy it to the root of your repository and commit it:

```bash
curl -fsSL -o AGENTS.md https://raw.githubusercontent.com/computesphere-samples/learn/main/agent-kit/AGENTS.md
```

`AGENTS.md` is a shared convention that several coding agents read at the start of every session. If yours reads a different file, point that file at this one or paste the sections in. Claude Code reads `CLAUDE.md`, which can import the file with a line containing `@AGENTS.md`.

Already have an `AGENTS.md`? Add this one's sections to it under a `## Deploying to ComputeSphere` heading. Then add your project's specifics: the service name, its port and health path, and how to run the tests.

## Use the skill

A skill is loaded only when its task comes up, so it costs no context the rest of the time. In Claude Code, a skill is a folder containing a `SKILL.md`:

```bash
mkdir -p .claude/skills/deploy-to-computesphere
curl -fsSL -o .claude/skills/deploy-to-computesphere/SKILL.md \
  https://raw.githubusercontent.com/computesphere-samples/learn/main/agent-kit/SKILL.md
```

Use `~/.claude/skills/` instead to have it in every project. Other agents call the same idea custom commands, prompt files or rules; the file is plain Markdown with a `name` and a `description`, so it ports with little change.

## Connect the MCP server

The server is at `https://mcp.computesphere.com/mcp` and speaks streamable HTTP. Authentication is OAuth: the first time your agent uses it, your browser opens to sign in to ComputeSphere. Opening the URL directly returns `401`. Full details: [MCP server docs](https://docs.computesphere.com/docs/integrations/mcp/).

| Agent | Setup |
| --- | --- |
| **Claude** | Add `https://mcp.computesphere.com/mcp` as a custom connector. See Anthropic's guide: [building custom connectors via remote MCP servers](https://support.claude.com/en/articles/11503834-building-custom-connectors-via-remote-mcp-servers) |
| **Cursor** | Use **Add to Cursor** on the [MCP server docs](https://docs.computesphere.com/docs/integrations/mcp/) page, or add the JSON below to `~/.cursor/mcp.json`. [Cursor's MCP docs](https://cursor.com/docs/context/mcp) |
| **ChatGPT** | Add the server URL by hand. See [OpenAI's MCP docs](https://platform.openai.com/docs/mcp) |
| **Claude Desktop** | Add the JSON below to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows) |
| **Windsurf** | Add the JSON below to `~/.codeium/windsurf/mcp_config.json` |
| **Zed** | Add it to `~/.config/zed/settings.json`, under `"context_servers"` |
| **VS Code** | Use `code --add-mcp` (below), or add it to the workspace's `.vscode/mcp.json`. That file lists servers under `"servers"`, not `"mcpServers"`: see [VS Code's MCP docs](https://code.visualstudio.com/docs/copilot/customization/mcp-servers) |
| **Continue** | Add it to `~/.continue/config.json`, under `"mcpServers"` |

The JSON, from the docs (Zed and VS Code nest it differently, as noted above):

```json
{
  "mcpServers": {
    "computesphere": {
      "url": "https://mcp.computesphere.com/mcp"
    }
  }
}
```

Restart your agent after editing its config.

Two agents also have a one-line command. These come from each agent's own CLI rather than the ComputeSphere docs:

```bash
# Claude Code
claude mcp add --transport http computesphere https://mcp.computesphere.com/mcp

# VS Code
code --add-mcp '{"name":"computesphere","type":"http","url":"https://mcp.computesphere.com/mcp"}'
```

### What it can do, and who approves it

The server's tools mirror the ComputeSphere API: list projects, services and deployments; read status and build, deploy and runtime logs; and make changes such as applying a manifest, restarting or rolling back. For deletions, applying a manifest, and replacing a whole set of variables or secrets, the server asks for a confirmation of that exact action, which you approve. Your agent sees the current tool list when it connects.

It acts **as you**, with everything your account can reach. That's handy for reading logs while you debug. For an agent that changes things unattended, use `csph` with a token limited to one project and a short expiry instead: see [API tokens](https://docs.computesphere.com/docs/getting-started/api-key/) and [`csph auth token create`](https://docs.computesphere.com/docs/cli/auth/token/create/).

## Learn the why

Path 4 of [ComputeSphere Learn](https://learn.computesphere.com), *Vibe coding to production*, teaches everything in these files to people: instruction files and skills, least-privilege tokens, deploying with an agent, and operating a live app with one.
