# jscan

Code quality analyzer for JavaScript/TypeScript projects.

## Installation

```bash
npm install -g jscan
```

Or use with npx:

```bash
npx jscan analyze src/
```

## Usage

```bash
# Analyze a directory
jscan analyze ./src

# Output as JSON
jscan analyze ./src --format json

# Output as HTML report
jscan analyze ./src --format html --output report.html
```

## Features

- Dead code detection
- Cyclomatic complexity analysis
- Duplicate code detection
- And more...

## Documentation

For full documentation, visit [GitHub](https://github.com/ludo-technologies/polyscan).

## License

MIT
