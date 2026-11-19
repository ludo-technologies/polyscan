package analyzer

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/ludo-technologies/polyscan/polyscan/internal/js/domain"
)

// nextSegmentExports are the exports that Next.js reads from every App
// Router file convention: the default export, the metadata and viewport
// API, and the route segment config.
var nextSegmentExports = []string{
	"default",
	"generateMetadata", "metadata", "generateViewport", "viewport",
	"generateStaticParams", "dynamic", "dynamicParams", "revalidate",
	"fetchCache", "runtime", "preferredRegion", "maxDuration",
}

// nextImageExports are the exports that Next.js reads from a generated
// image file, such as opengraph-image.tsx, in addition to nextSegmentExports.
var nextImageExports = []string{"alt", "size", "contentType", "generateImageMetadata"}

// nextAppFileExports maps each App Router file convention to the exports
// that Next.js reads from it in addition to nextSegmentExports. A key is a
// file name without its extension, and the file must be under an app
// directory.
var nextAppFileExports = map[string][]string{
	"page":             nil,
	"layout":           nil,
	"template":         nil,
	"loading":          nil,
	"error":            nil,
	"global-error":     nil,
	"not-found":        nil,
	"global-not-found": nil,
	"forbidden":        nil,
	"unauthorized":     nil,
	"default":          nil,
	"route":            {"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
	"robots":           nil,
	"manifest":         nil,
	"sitemap":          {"generateSitemaps"},
	"icon":             nextImageExports,
	"apple-icon":       nextImageExports,
	"opengraph-image":  nextImageExports,
	"twitter-image":    nextImageExports,
}

// nextRootFileExports maps each file convention that Next.js looks for in
// the directory holding the app or pages directory to the exports that it
// reads from that file. A key is a file name without its extension.
var nextRootFileExports = map[string][]string{
	"middleware":             {"default", "middleware", "config"},
	"proxy":                  {"default", "proxy", "config"},
	"instrumentation":        {"register", "onRequestError"},
	"instrumentation-client": {"onRouterTransitionStart"},
}

// nextProjectRoots returns the directories that hold an app or pages
// directory among the analyzed files. Next.js looks for the files in
// nextRootFileExports in these directories.
func nextProjectRoots(allModuleInfos map[string]*domain.ModuleInfo) map[string]bool {
	roots := make(map[string]bool)
	for filePath := range allModuleInfos {
		for dir := filepath.Dir(filePath); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			switch filepath.Base(dir) {
			case "app", "pages":
				roots[filepath.Dir(dir)] = true
			}
		}
	}
	return roots
}

// isNextReservedExport reports whether Next.js reads exportedName from
// filePath because of the file's name and location, in which case no
// import of the export is expected. nextRoots comes from nextProjectRoots.
func isNextReservedExport(filePath, exportedName string, nextRoots map[string]bool) bool {
	base := filepath.Base(filePath)
	convention := strings.TrimSuffix(base, filepath.Ext(base))
	if extra, ok := nextAppFileExports[convention]; ok && strings.Contains(filepath.ToSlash(filePath), "/app/") {
		return slices.Contains(nextSegmentExports, exportedName) || slices.Contains(extra, exportedName)
	}
	return nextRoots[filepath.Dir(filePath)] && slices.Contains(nextRootFileExports[convention], exportedName)
}
