import { joinSession } from "@github/copilot-sdk/extension";
import { execFile } from "node:child_process";
import { readFile } from "node:fs/promises";
import { join } from "node:path";

/**
 * Documentation Check Extension
 *
 * Provides a tool that maps code changes to affected documentation,
 * helping agents maintain docs/game/ and docs/dev/ in sync with code.
 */

async function loadDocumentationMap(cwd) {
    const mapPath = join(cwd, "docs", "documentation-map.yaml");
    const content = await readFile(mapPath, "utf-8");
    return parseYaml(content);
}

/**
 * Minimal YAML parser for the documentation-map.yaml format.
 * Handles the specific structure we use (mappings array + excluded_paths).
 */
function parseYaml(content) {
    const mappings = [];
    const excludedPaths = [];
    const lines = content.split("\n");

    let inMappings = false;
    let inExcluded = false;
    let currentMapping = null;
    let currentDocList = null;
    let currentDoc = null;

    for (const rawLine of lines) {
        const line = rawLine.trimEnd();

        if (line.startsWith("#") || line.trim() === "") continue;

        if (line === "mappings:") {
            inMappings = true;
            inExcluded = false;
            continue;
        }
        if (line === "excluded_paths:") {
            inMappings = false;
            inExcluded = true;
            if (currentMapping) {
                mappings.push(currentMapping);
                currentMapping = null;
            }
            continue;
        }

        if (inExcluded) {
            const match = line.match(/^\s+-\s+"?([^"]+)"?\s*$/);
            if (match) {
                excludedPaths.push(match[1]);
            }
            continue;
        }

        if (inMappings) {
            // New mapping entry
            if (line.match(/^\s{2}-\s+code_path:/)) {
                if (currentMapping) mappings.push(currentMapping);
                const val = line.replace(/^\s{2}-\s+code_path:\s*/, "").trim();
                currentMapping = { code_path: val, dev_docs: [], game_docs: [], notes: "" };
                currentDocList = null;
                currentDoc = null;
                continue;
            }

            if (!currentMapping) continue;

            // dev_docs or game_docs list start
            if (line.match(/^\s{4}dev_docs:/)) {
                currentDocList = "dev_docs";
                currentDoc = null;
                continue;
            }
            if (line.match(/^\s{4}game_docs:/)) {
                if (line.includes("[]")) {
                    currentMapping.game_docs = [];
                    currentDocList = null;
                } else {
                    currentDocList = "game_docs";
                }
                currentDoc = null;
                continue;
            }

            // notes field
            const notesMatch = line.match(/^\s{4}notes:\s*(.*)$/);
            if (notesMatch) {
                currentMapping.notes = notesMatch[1];
                currentDocList = null;
                continue;
            }

            // Doc entry (path)
            if (currentDocList) {
                const pathMatch = line.match(/^\s{6,8}-\s+path:\s*(.+)$/);
                if (pathMatch) {
                    currentDoc = { path: pathMatch[1].trim(), confidence: "directly_affected" };
                    currentMapping[currentDocList].push(currentDoc);
                    continue;
                }
                const confMatch = line.match(/^\s{8,10}confidence:\s*(.+)$/);
                if (confMatch && currentDoc) {
                    currentDoc.confidence = confMatch[1].trim();
                    continue;
                }
            }
        }
    }

    if (currentMapping) mappings.push(currentMapping);
    return { mappings, excludedPaths };
}

function getChangedFiles(cwd, baseBranch) {
    return new Promise((resolve, reject) => {
        execFile(
            "git",
            ["diff", "--name-only", `$(git merge-base HEAD ${baseBranch})`, "HEAD"],
            { cwd, shell: true },
            (err, stdout, stderr) => {
                if (err) {
                    // Fallback: try diffing against baseBranch directly
                    execFile(
                        "git",
                        ["diff", "--name-only", baseBranch, "HEAD"],
                        { cwd, shell: true },
                        (err2, stdout2) => {
                            if (err2) reject(new Error(`git diff failed: ${stderr || err2.message}`));
                            else resolve(stdout2.trim().split("\n").filter(Boolean));
                        }
                    );
                } else {
                    resolve(stdout.trim().split("\n").filter(Boolean));
                }
            }
        );
    });
}

function isExcluded(filePath, excludedPaths) {
    for (const pattern of excludedPaths) {
        if (pattern.startsWith("*.")) {
            const ext = pattern.slice(1);
            if (filePath.endsWith(ext)) return true;
        } else if (pattern.endsWith("/")) {
            if (filePath.startsWith(pattern)) return true;
        } else {
            if (filePath === pattern || filePath.startsWith(pattern)) return true;
        }
    }
    return false;
}

function checkDocumentationImpact(changedFiles, map) {
    const { mappings, excludedPaths } = map;
    const results = {
        covered: [],
        missing: [],
        unmapped: [],
        summary: "",
    };

    const changedDocsSet = new Set(changedFiles.filter((f) => f.startsWith("docs/")));
    const codeFiles = changedFiles.filter((f) => !isExcluded(f, excludedPaths));

    for (const file of codeFiles) {
        let matched = false;

        for (const mapping of mappings) {
            if (file.startsWith(mapping.code_path)) {
                matched = true;
                const allDocs = [...mapping.dev_docs, ...mapping.game_docs];

                for (const doc of allDocs) {
                    const entry = {
                        code_file: file,
                        doc_path: doc.path,
                        confidence: doc.confidence,
                        code_path_rule: mapping.code_path,
                        notes: mapping.notes,
                    };

                    if (changedDocsSet.has(doc.path)) {
                        results.covered.push(entry);
                    } else {
                        results.missing.push(entry);
                    }
                }
                break; // Use most specific match (first match)
            }
        }

        if (!matched) {
            results.unmapped.push(file);
        }
    }

    // Deduplicate missing by doc_path
    const seenMissing = new Set();
    results.missing = results.missing.filter((entry) => {
        if (seenMissing.has(entry.doc_path)) return false;
        seenMissing.add(entry.doc_path);
        return true;
    });

    const seenCovered = new Set();
    results.covered = results.covered.filter((entry) => {
        if (seenCovered.has(entry.doc_path)) return false;
        seenCovered.add(entry.doc_path);
        return true;
    });

    // Build summary
    const parts = [];
    if (results.covered.length > 0) {
        parts.push(`✅ ${results.covered.length} doc(s) already updated`);
    }
    if (results.missing.length > 0) {
        parts.push(`⚠️  ${results.missing.length} doc(s) may need updating`);
    }
    if (results.unmapped.length > 0) {
        parts.push(`🔍 ${results.unmapped.length} file(s) unmapped — manual review required`);
    }
    if (parts.length === 0) {
        parts.push("No documentation impact detected (all changes in excluded paths).");
    }
    results.summary = parts.join("\n");

    return results;
}

function formatReport(results) {
    let report = `## Documentation Impact Report\n\n${results.summary}\n`;

    if (results.missing.length > 0) {
        report += "\n### ⚠️  Docs That May Need Updating\n\n";
        for (const entry of results.missing) {
            const conf = entry.confidence === "directly_affected" ? "🔴 directly affected" : "🟡 possibly affected";
            report += `- **${entry.doc_path}** (${conf})\n`;
            report += `  - Triggered by: \`${entry.code_path_rule}\`\n`;
            if (entry.notes) report += `  - Context: ${entry.notes}\n`;
        }
    }

    if (results.covered.length > 0) {
        report += "\n### ✅ Docs Already Updated\n\n";
        for (const entry of results.covered) {
            report += `- ${entry.doc_path}\n`;
        }
    }

    if (results.unmapped.length > 0) {
        report += "\n### 🔍 Unmapped Files (Manual Review Required)\n\n";
        for (const file of results.unmapped) {
            report += `- \`${file}\`\n`;
        }
        report += "\nThese files have no mapping in `docs/documentation-map.yaml`. ";
        report += "Check if they introduce behavior that should be documented.\n";
    }

    return report;
}

// ─── Extension Entry Point ───

const session = await joinSession({
    hooks: {},
    tools: [
        {
            name: "check_documentation_impact",
            description:
                "Analyzes code changes on the current branch and reports which documentation files (docs/game/ and docs/dev/) may need updating. Uses docs/documentation-map.yaml as the mapping source. Call this before finishing a branch or creating a PR.",
            parameters: {
                type: "object",
                properties: {
                    base_branch: {
                        type: "string",
                        description:
                            "The base branch to diff against (default: 'main'). Used to determine which files changed.",
                        default: "main",
                    },
                },
                required: [],
            },
            handler: async (args, invocation) => {
                const cwd = process.cwd();
                const baseBranch = args.base_branch || "main";

                try {
                    const map = await loadDocumentationMap(cwd);
                    const changedFiles = await getChangedFiles(cwd, baseBranch);

                    if (changedFiles.length === 0) {
                        return "No changed files detected compared to " + baseBranch + ".";
                    }

                    const results = checkDocumentationImpact(changedFiles, map);
                    return formatReport(results);
                } catch (err) {
                    return `Error: ${err.message}\n\nMake sure you're in the repository root and docs/documentation-map.yaml exists.`;
                }
            },
        },
    ],
});
