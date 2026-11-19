package analyzer

import (
	"slices"
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
)

// nextFixture holds two Next.js projects: /web keeps an App Router under
// src, and /docs keeps a Pages Router at the project root.
var nextFixture = map[string]string{
	"/web/src/app/page.tsx": `export default function Home() { return null; }`,
	"/web/src/app/robots.ts": `
export default function robots() { return {}; }
export function helper() {}
`,
	"/web/src/app/sitemap.ts": `
export function generateSitemaps() { return []; }
export default function sitemap() { return []; }
`,
	"/web/src/app/manifest.ts": `export default function manifest() { return {}; }`,
	"/web/src/app/blog/opengraph-image.tsx": `
export const alt = "Blog";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export function generateImageMetadata() { return []; }
export default function Image() { return null; }
`,
	"/web/src/app/icon.tsx":         `export default function Icon() { return null; }`,
	"/web/src/app/global-error.tsx": `export default function GlobalError() { return null; }`,
	"/web/src/app/forbidden.tsx":    `export default function Forbidden() { return null; }`,
	"/web/src/app/unauthorized.tsx": `export default function Unauthorized() { return null; }`,
	"/web/src/app/global-not-found.tsx": `
export const metadata = {};
export default function GlobalNotFound() { return null; }
`,
	"/web/src/middleware.ts": `
export default function middleware() {}
export const config = { matcher: "/" };
`,
	"/web/src/instrumentation.ts": `
export function register() {}
export function onRequestError() {}
`,
	"/web/src/instrumentation-client.ts": `export function onRouterTransitionStart() {}`,
	// Next.js looks for middleware next to the app directory, which is in
	// src, so neither of these files is a Next.js convention file.
	"/web/middleware.ts": `export default function middleware() {}`,
	"/web/src/lib/middleware.ts": `
export default function middleware() {}
export const config = {};
`,
	"/docs/pages/index.tsx": `export default function Home() { return null; }`,
	"/docs/proxy.ts": `
export function proxy() {}
export const config = { matcher: "/" };
`,
}

func analyzeNextFixture(t *testing.T) (map[string]*domain.ModuleInfo, *ImportGraph) {
	t.Helper()
	allInfos := make(map[string]*domain.ModuleInfo, len(nextFixture))
	analyzedFiles := make(map[string]bool, len(nextFixture))
	for path, source := range nextFixture {
		_, info := parseAndAnalyzeTSNamed(t, path, source)
		allInfos[path] = info
		analyzedFiles[path] = true
	}
	return allInfos, BuildImportGraph(allInfos, analyzedFiles)
}

func findingDescriptions(findings []*DeadCodeFinding) []string {
	descriptions := make([]string, 0, len(findings))
	for _, f := range findings {
		descriptions = append(descriptions, f.FilePath+": "+f.Description)
	}
	slices.Sort(descriptions)
	return descriptions
}

func TestIsNextReservedExport(t *testing.T) {
	allInfos, _ := analyzeNextFixture(t)
	nextRoots := nextProjectRoots(allInfos)

	tests := []struct {
		filePath string
		name     string
		want     bool
	}{
		{"/web/src/app/page.tsx", "default", true},
		{"/web/src/app/page.tsx", "generateMetadata", true},
		{"/web/src/app/page.tsx", "generateSitemaps", false},
		{"/web/src/app/api/route.ts", "GET", true},
		{"/web/src/app/sitemap.ts", "generateSitemaps", true},
		{"/web/src/app/sitemap.ts", "revalidate", true},
		{"/web/src/app/blog/opengraph-image.tsx", "alt", true},
		{"/web/src/app/blog/opengraph-image.tsx", "size", true},
		{"/web/src/app/blog/opengraph-image.tsx", "contentType", true},
		{"/web/src/app/blog/opengraph-image.tsx", "generateImageMetadata", true},
		{"/web/src/app/robots.ts", "helper", false},
		{"/web/src/lib/robots.ts", "default", false},
		{"/web/src/middleware.ts", "default", true},
		{"/web/src/middleware.ts", "middleware", true},
		{"/web/src/middleware.ts", "config", true},
		{"/web/src/middleware.ts", "proxy", false},
		{"/web/src/instrumentation.ts", "register", true},
		{"/web/src/instrumentation.ts", "onRequestError", true},
		{"/web/src/instrumentation-client.ts", "onRouterTransitionStart", true},
		{"/web/src/instrumentation-client.ts", "register", false},
		{"/web/middleware.ts", "default", false},
		{"/web/src/lib/middleware.ts", "config", false},
		{"/docs/proxy.ts", "proxy", true},
		{"/docs/proxy.ts", "config", true},
		{"/docs/proxy.ts", "middleware", false},
	}
	for _, tt := range tests {
		if got := isNextReservedExport(tt.filePath, tt.name, nextRoots); got != tt.want {
			t.Errorf("isNextReservedExport(%q, %q) = %v, want %v", tt.filePath, tt.name, got, tt.want)
		}
	}
}

func TestDetectUnusedExportedFunctions_NextConventionFiles(t *testing.T) {
	allInfos, graph := analyzeNextFixture(t)

	got := findingDescriptions(DetectUnusedExportedFunctions(allInfos, graph))
	want := []string{
		"/web/middleware.ts: Exported function 'default' is not imported by any other analyzed file",
		"/web/src/app/robots.ts: Exported function 'helper' is not imported by any other analyzed file",
		"/web/src/lib/middleware.ts: Exported function 'default' is not imported by any other analyzed file",
	}
	if !slices.Equal(got, want) {
		t.Errorf("findings:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
