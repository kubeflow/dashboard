'use strict';

const {resolve} = require('path');
const {execSync} = require('child_process');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const DefinePlugin = require('webpack').DefinePlugin;
const ESLintPlugin = require('eslint-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const TerserPlugin = require('terser-webpack-plugin');
let commit = process.env.BUILD_COMMIT || '';

try {
    commit = commit || `${execSync(`git rev-parse HEAD`)}`.replace(/\s/g, '');
} catch (e) {}

const ENV = process.env.NODE_ENV || 'development';
const NODE_MODULES = /\/node_modules\//;
const PKG_VERSION =
    `${require('./package.json').version}-${commit.slice(0, 6)}`;
const BUILD_VERSION = process.env.BUILD_VERSION || `dev_local`;
const SRC = resolve(__dirname, 'public');
const COMPONENTS = resolve(SRC, 'components');
const DESTINATION = resolve(__dirname, 'dist', 'public');
const WEBCOMPONENTS = resolve(
    __dirname, 'node_modules', '@webcomponents', 'webcomponentsjs');
const POLYFILLS = [
    {
        from: '*.{js,map}',
        to: resolve(DESTINATION, 'webcomponentsjs', '[name][ext]'),
        context: WEBCOMPONENTS,
    },
    {
        from: 'bundles/*.{js,map}',
        to: resolve(DESTINATION, 'webcomponentsjs', 'bundles', '[name][ext]'),
        context: WEBCOMPONENTS,
    },
];

/**
 * Rules for processing Polymer components to allow external Pug templates
 * and CSS files.
 */
const COMPONENT_RULES = [
    {
        test: /\.pug$/,
        use: ['pug-loader'],
    },
    {
        test: /\.css$/,
        include: COMPONENTS,
        use: [
            {
                loader: 'css-loader',
                options: {
                    exportType: 'string',
                },
            },
        ],
    },
];

module.exports = {
    mode: ENV,
    entry: {
        app: resolve(SRC, 'index.js'),
        dashboard_lib: resolve(SRC, 'library.js'),
    },
    output: {
        filename: '[name].bundle.js',
        path: DESTINATION,
        library: 'centraldashboard',
        libraryTarget: 'umd',
    },
    devtool: 'cheap-source-map',
    module: {
        rules: COMPONENT_RULES.concat([
            {
                test: /\.css$/,
                exclude: COMPONENTS,
                use: [
                    MiniCssExtractPlugin.loader,
                    'css-loader',
                ],
            },
            {
                test: /\.(gif|ico|jpg|png)$/,
                type: 'asset/resource',
                generator: {
                    filename: 'assets/[name][ext]',
                },
            },
            {
                test: /\.svg$/,
                type: 'asset/source',
            },
            // Roboto Font and Material Icons
            {
                test: /(iconfont|roboto)\/.*\.(eot|svg|ttf|woff2?)$/,
                type: 'asset',
                parser: {
                    dataUrlCondition: {
                        maxSize: 8192,
                    },
                },
                generator: {
                    filename: 'fonts/[name][ext]',
                },
            },
            {
                test: /\.js$/,
                exclude: NODE_MODULES,
                use: {
                    loader: 'babel-loader',
                    options: {
                        cacheDirectory: true,
                        presets: [[
                            '@babel/preset-env',
                            {
                                corejs: '2',
                                useBuiltIns: 'entry',
                                targets: {
                                    browsers: [
                                        // Best practice: https://github.com/babel/babel/issues/7789
                                        '>=1%',
                                        'not ie 11',
                                        'not op_mini all',
                                    ],
                                },
                            },
                        ]],
                        plugins: ['@babel/plugin-transform-runtime'],
                    },
                },
            },
        ]),
    },
    optimization: {
        minimizer: [
          new TerserPlugin({
            parallel: true,
            extractComments: true,
          })
        ],
    },
    plugins: [
        new CleanWebpackPlugin(),
        new CopyWebpackPlugin({
            patterns: POLYFILLS.concat(
              [{from: resolve(SRC, 'kubeflow-palette.css'), to: DESTINATION}]
            )
        }),
        new DefinePlugin({
            BUILD_VERSION: JSON.stringify(BUILD_VERSION),
            VERSION: JSON.stringify(PKG_VERSION),
        }),
        new ESLintPlugin({
            extensions: ['js'],
            exclude: ['node_modules'],
            failOnError: true,
            fix: true,
        }),
        new HtmlWebpackPlugin({
            filename: resolve(DESTINATION, 'index.html'),
            template: resolve(SRC, 'index.html'),
            inject: true,
            scriptLoading: 'defer',
            excludeChunks: ['dashboard_lib'],
            minify: ENV == 'development' ? false : {
                collapseWhitespace: true,
                removeComments: true,
                removeRedundantAttributes: true,
                removeScriptTypeAttributes: true,
                removeStyleLinkTypeAttributes: true,
                useShortDoctype: true,
            },
        }),
        new MiniCssExtractPlugin({
            filename: '[name].css',
            chunkFilename: '[id].css',
        }),
    ],
    devServer: {
        port: 8080,
        proxy: {
            '/api': {
                target: 'http://localhost:8082',
            },
            '/jupyter': {
                target: 'http://localhost:8085',
                pathRewrite: {'^/jupyter': ''},
            },
            // NOTE: this makes `/notebook` requests fail with a 504 error
            '/notebook': {
                target: 'http://localhost:8086',
                pathRewrite: {
                    '^/notebook/(.*?)/(.*?)/(.*?)':
                        '/$1/services/$2/proxy/notebook/$1/$2/$3',
                },
            },
            '/pipeline': {
                target: 'http://localhost:8087',
                pathRewrite: {'^/pipeline': ''},
            },
        },
        historyApiFallback: {
            disableDotRule: true,
        },
    },
};
