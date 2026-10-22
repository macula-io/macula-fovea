"""Generate macula-fovea's two C4 diagrams into assets/.

Run:  python3 architecture/c4_diagrams.py   (from the repository root)

macula-fovea is the fovea spec (spec/) and its reference implementation, the
fovea CLI (cli/cmd/fovea, one static Go binary), which the GitHub Action
(action.yml) builds and runs in a consumer's CI. The diagrams draw that tool:

  c4-context.svg     Level 1: fovea, who runs it and what it reads and writes
  c4-components.svg  Level 3: inside the fovea binary (cli/internal/core)

Every element is read from the code: cli/, action.yml, scripts/ and the
workflow. Colours are plain classes with a prefers-color-scheme: dark block,
no CSS custom properties (rsvg-convert renders those black), the same style
and legend as mcl-fovea's diagrams. After a change, render each
(rsvg-convert -w 1400 assets/c4-context.svg -o /tmp/c.png) and look at it;
keep architecture/README.md's tables in step.
"""
import os
from xml.sax.saxutils import escape

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "assets")

SANS = '"Atkinson Hyperlegible", "Segoe UI", system-ui, -apple-system, Helvetica, Arial, sans-serif'
MONO = '"JetBrains Mono", ui-monospace, Menlo, Consolas, monospace'

# Element kinds, each (light fill, light stroke, dark fill, dark stroke).
KINDS = {
    "b-sys": ("#F5E9CC", "#A56F0A", "#3A2E13", "#E3AA3C"),   # in scope
    "b-per": ("#E3EBF2", "#3B5B7A", "#1D2A36", "#8EB4D8"),   # person
    "b-mac": ("#E6E4EE", "#5E5A78", "#26243A", "#A9A4C8"),   # Macula infrastructure
    "b-ext": ("#ECEFED", "#8A9791", "#1C2421", "#7D8B85"),   # external system
    "b-cmp": ("#FFFFFF", "#8A9791", "#171F1C", "#7D8B85"),   # component or container
}

STYLE = f"""<style>
.t-name {{ font-family: {SANS}; font-size: 15px; font-weight: 700; }}
.t-desc {{ font-family: {SANS}; font-size: 12.5px; }}
.t-rel {{ font-family: {SANS}; font-size: 11.5px; paint-order: stroke; stroke-width: 4px; stroke-linejoin: round; }}
.t-code {{ font-family: {MONO}; font-size: 13px; font-weight: 600; }}
.t-type {{ font-family: {MONO}; font-size: 11.5px; }}
.t-bound {{ font-family: {MONO}; font-size: 12px; font-weight: 600; }}
.t-leg {{ font-family: {SANS}; font-size: 12px; }}
.edge {{ fill: none; stroke-width: 1.25; }}
.edge-later {{ fill: none; stroke-width: 1.25; stroke-dasharray: 5 4; }}
.bg {{ fill: #FFFFFF; }}
""" + "".join(f".{k} {{ fill: {v[0]}; stroke: {v[1]}; stroke-width: {2 if k == 'b-sys' else 1.25}; }}\n" for k, v in KINDS.items()) + """\
.b-later { fill: none; stroke: #8A9791; stroke-width: 1.25; stroke-dasharray: 5 4; }
.bound { fill: none; stroke: #B3BDB8; stroke-width: 1; }
.head { fill: #3B5B7A; }
.t-name, .t-code { fill: #16201D; }
.t-type, .t-bound, .t-leg { fill: #66736E; }
.t-desc { fill: #34413C; }
.t-rel { fill: #66736E; stroke: #FFFFFF; }
.edge, .edge-later { stroke: #8A9791; }
.arrowhead { fill: #8A9791; }
@media (prefers-color-scheme: dark) {
.bg { fill: #171F1C; }
""" + "".join(f".{k} {{ fill: {v[2]}; stroke: {v[3]}; }}\n" for k, v in KINDS.items()) + """\
.b-later { stroke: #7D8B85; }
.bound { stroke: #4A5752; }
.head { fill: #8EB4D8; }
.t-name, .t-code { fill: #E6ECE9; }
.t-type, .t-bound, .t-leg { fill: #93A09A; }
.t-desc { fill: #C3CDC8; }
.t-rel { fill: #93A09A; stroke: #171F1C; }
.edge, .edge-later { stroke: #7D8B85; }
.arrowhead { fill: #7D8B85; }
}
</style>"""

DEFS = ('<defs><marker id="a" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" '
        'orient="auto-start-reverse"><path class="arrowhead" d="M0 0L10 5L0 10z"/></marker></defs>')

LEGEND = [("b-sys", "In scope"), ("b-per", "Person"), ("b-mac", "Macula infrastructure"),
          ("b-ext", "External system"), ("b-cmp", "Container or component"), ("b-later", "Planned, not built")]
BOUNDARY = ("bound", "Boundary")


class Diagram:
    def __init__(self, name, title, w, h):
        self.name, self.title, self.w, self.h, self.parts = name, title, w, h, []

    def text(self, cls, x, y, s, anchor=None, rotate=None):
        a = f' text-anchor="{anchor}"' if anchor else ""
        r = f' transform="rotate({rotate} {x} {y})"' if rotate is not None else ""
        self.parts.append(f'<text class="{cls}" x="{x}" y="{y}"{a}{r}>{escape(s)}</text>')

    def box(self, cls, x, y, w, h, name, kind, desc, person=False, code=False):
        self.parts.append(f'<rect class="{cls}" x="{x}" y="{y}" width="{w}" height="{h}" rx="10"/>')
        tx = x + 16
        if person:
            self.parts.append(f'<circle class="head" cx="{x + 26}" cy="{y + 24}" r="9"/>')
            tx = x + 44
        self.text("t-code" if code else "t-name", tx, y + 28, name)
        self.text("t-type", tx, y + 46, kind)
        for i, line in enumerate(desc):
            self.text("t-desc", x + 16, y + 70 + 16 * i, line)

    def bound(self, x, y, w, h, label):
        self.parts.append(f'<rect class="bound" x="{x}" y="{y}" width="{w}" height="{h}" rx="12"/>')
        self.text("t-bound", x + 16, y + 22, label)

    def edge(self, d, later=False):
        self.parts.append(f'<path class="{"edge-later" if later else "edge"}" d="{d}" marker-end="url(#a)"/>')

    def legend(self, y, boundary=True):
        x = 24
        for cls, label in LEGEND + ([BOUNDARY] if boundary else []):
            self.parts.append(f'<rect class="{cls}" x="{x}" y="{y}" width="22" height="14" rx="3"/>')
            self.text("t-leg", x + 30, y + 12, label)
            x += 34 + 6.6 * len(label)

    def write(self):
        svg = [(f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {self.w} {self.h}" role="img" '
                f'aria-label="{escape(self.title)}">'), f"<title>{escape(self.title)}</title>", STYLE, DEFS,
               f'<rect class="bg" x="0" y="0" width="{self.w}" height="{self.h}"/>'] + self.parts + ["</svg>", ""]
        with open(os.path.join(OUT, self.name), "w") as f:
            f.write("\n".join(svg))



def context():
    d = Diagram("c4-context.svg", "fovea system context", 1000, 630)
    d.box("b-per", 40, 30, 270, 96, "Assessment author", "[Person]",
          ["Writes the header and the cells;", "runs init, lint, score, render"], person=True)
    d.box("b-ext", 365, 30, 270, 96, "Consumer CI", "[External: GitHub Actions]",
          ["Runs action.yml pinned by sha:", "lint annotations, scorecard"])
    d.box("b-per", 690, 30, 270, 96, "Anyone who checks", "[Person]",
          ["A buyer, partner or auditor:", "runs fovea verify offline"], person=True)
    d.box("b-ext", 40, 200, 270, 120, "Assessment repository", "[External: git]",
          ["fovea.yaml, landscape, anchors,", "one YAML file per cell; its", "history holds every revision"])
    d.box("b-sys", 365, 200, 270, 120, "fovea", "[CLI, one static Go binary]",
          ["The spec's reference tool: init,", "lint, score, render, issues and", "verify (--chain, --json)"])
    d.box("b-ext", 690, 200, 270, 120, "GitHub REST API", "[External system]",
          ["Issues for roadmap and overdue", "cells; this repo's releases for", "the Action's pin check"])
    d.box("b-mac", 40, 430, 270, 120, "Realm public key", "[Macula infrastructure]",
          ["The trust anchor: an observer's", "endorsement and every record", "are checked against it"])
    d.box("b-ext", 365, 430, 270, 120, "Observation records", "[External: files]",
          ["Signed 0x23 claim observations", "and endorsements, e.g. a clone", "of mcl-fovea-records"])
    d.box("b-ext", 690, 430, 270, 120, "mcl-fovea observer", "[External: ours, mesh service]",
          ["Probes, judges, signs and chains", "each verdict; implements the", "same rules as this lint"])
    d.edge("M175 126V200"); d.text("t-rel", 183, 162, "writes cells")
    d.edge("M310 100H338V230H365"); d.text("t-rel", 346, 170, "runs", rotate=90)
    d.edge("M500 126V200"); d.text("t-rel", 508, 162, "builds and runs it")
    d.edge("M690 100H662V230H635"); d.text("t-rel", 656, 165, "runs verify", anchor="middle", rotate=-90)
    d.edge("M365 280H310"); d.text("t-rel", 337, 270, "reads,", anchor="middle"); d.text("t-rel", 337, 302, "git", anchor="middle")
    d.edge("M635 280H690"); d.text("t-rel", 662, 270, "HTTPS", anchor="middle"); d.text("t-rel", 662, 302, "REST", anchor="middle")
    d.edge("M500 320V430"); d.text("t-rel", 508, 368, "verify reads, offline")
    d.edge("M420 320V375H175V430"); d.text("t-rel", 250, 367, "checks against")
    d.edge("M690 490H635"); d.text("t-rel", 662, 480, "signs", anchor="middle"); d.text("t-rel", 662, 512, "records", anchor="middle")
    d.legend(590, boundary=False)
    d.write()


def components():
    d = Diagram("c4-components.svg", "fovea components", 1240, 800)
    d.box("b-cmp", 286, 30, 326, 100, "action.yml", "[composite GitHub Action]",
          ["Checks its own pin, builds fovea,", "runs it, annotates, writes artifacts"], code=True)
    d.box("b-cmp", 628, 30, 326, 100, "spec/v0.6/", "[the spec, normative]",
          ["Axes, attributes, cell schema,", "scorecard, observations, probes"], code=True)
    d.bound(270, 170, 700, 540, "fovea [cli/, one static binary]")
    d.box("b-sys", 286, 205, 668, 80, "cmd/fovea main.go", "[entry point]",
          ["Parses the command line; dispatches; --version names the tool and the spec versions it reads."], code=True)
    for i, (name, kind, a, b) in enumerate([
            ("init", "[init.go]", "Cell skeletons", "from the header"),
            ("lint", "[lint.go, rules.go]", "Grid, status, content,", "anti-theater rules"),
            ("score, render", "[score, render]", "Scorecard; md,", "html or json"),
            ("issues", "[issues, github]", "Roadmap and overdue", "cells to issues")]):
        d.box("b-cmp", 286 + i * 171, 310, 155, 110, name, kind, [a, b], code=True)
    d.box("b-cmp", 286, 450, 326, 110, "types.go", "[LoadHeader, LoadCells]",
          ["Reads fovea.yaml and every cell", "file; spec versions 0.2 to 0.6"], code=True)
    d.box("b-cmp", 628, 450, 326, 110, "verify", "[verify.go, verify_chain.go]",
          ["Nine steps, offline; --chain reads", "C1 to C7; the assessment revision", "from git objects, never a checkout"], code=True)
    d.box("b-cmp", 286, 590, 326, 100, "gopkg.in/yaml.v3", "[library]", ["The header and cells"])
    d.box("b-mac", 628, 590, 326, 100, "macula-go v0.28.0", "[Macula SDK: cbor, identity, ...]",
          ["record, profile: the 0x23 record,", "ML-DSA signatures, the realm key"])
    d.box("b-ext", 20, 450, 200, 110, "Assessment", "[External: git]",
          ["fovea.yaml and", "cells, history"])
    d.box("b-ext", 1020, 310, 200, 110, "GitHub REST API", "[External]",
          ["Issues, marked", "fovea-cell: id"])
    d.box("b-ext", 1020, 450, 200, 110, "Observations", "[External: files]",
          ["Records, endorse-", "ments, realm key"])
    d.edge("M580 130V205"); d.text("t-rel", 588, 155, "builds, runs")
    d.edge("M791 130V205"); d.text("t-rel", 799, 155, "implements")
    for x in (363, 534, 705, 876):
        d.edge(f"M{x} 285V310")
    d.edge("M791 285V450"); d.text("t-rel", 799, 442, "verify")
    d.edge("M363 420V450"); d.edge("M534 420V450")
    d.edge("M705 420V435H600V450"); d.text("t-rel", 371, 442, "load")
    d.edge("M286 505H220"); d.text("t-rel", 245, 495, "reads", anchor="middle")
    d.edge("M449 560V590"); d.text("t-rel", 457, 580, "parses")
    d.edge("M791 560V590"); d.text("t-rel", 799, 580, "verifies with")
    d.edge("M954 365H1020"); d.text("t-rel", 987, 355, "HTTPS", anchor="middle")
    d.edge("M954 505H1020"); d.text("t-rel", 987, 495, "reads", anchor="middle")
    d.legend(760)
    d.write()


if __name__ == "__main__":
    for make in (context, components):
        make()
