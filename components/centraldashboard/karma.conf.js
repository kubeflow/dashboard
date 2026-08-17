const webpackConfig = require('./webpack.config');
webpackConfig.entry = ''; // Karma will supply the entry points
webpackConfig.devtool = 'inline-source-map';

// Add istanbul instrumentation via babel plugin for coverage
const babelRule = webpackConfig.module.rules.find(
    (rule) => rule.use && rule.use.loader === 'babel-loader'
);
if (babelRule) {
    babelRule.use.options.plugins = (babelRule.use.options.plugins || [])
        .concat(['istanbul']);
}

module.exports = (config) => config.set({
    basePath: '',
    browsers: ['ChromeHeadlessTest'],
    customLaunchers: {
        ChromeHeadlessTest: {
            base: 'ChromeHeadless',
            flags: ['--no-sandbox'],
        },
    },
    frameworks: ['jasmine', 'webpack'],
    files: [
        'public/index_test.js',
        /* Served-only fixtures for real iframe navigation tests. */
        {
            pattern: 'test_fixtures/*.html',
            included: false,
            served: true,
            watched: false,
        },
    ],
    exclude: [],
    preprocessors: {
        'public/index_test.js': ['webpack', 'sourcemap'],
    },
    webpack: webpackConfig,
    webpackMiddleware: {stats: 'errors-only'},
    reporters: ['progress', 'kjhtml'],
    coverageIstanbulReporter: {
        'reports': ['html', 'text'],
        'fixWebpackSourcePaths': true,
        'skipFilesWithNoCoverage': false,
        'report-config': {
            html: {
                subdir: 'public',
            },
        },
    },
});
