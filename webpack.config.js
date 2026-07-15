const path = require('path');

const tsLoaderConfig = {
    test: /\.ts$/i,
    use: 'ts-loader',
    exclude: '/node_modules/'
}

const tsxLoaderConfig = {
    test: /\.tsx$/i,
    use: 'ts-loader',
    exclude: '/node_modules/'
}

const cssLoaderConfig = {
    test: /\.css$/i,
    use: [
        'style-loader',
        'css-loader',
    ],
    exclude: '/node_modules/'
}

function baseConfig(rules = []) {
    if (rules.length === 0) {
        rules = [tsLoaderConfig, tsxLoaderConfig]
    }
    return {
        resolve: {
            extensions: ['.ts', '.tsx', '.css', '...'],
        },
        mode: 'production',
        module: {
            rules: [
                ...rules
            ]
        }
    }
}

module.exports = [
    {
        entry: './static_src/app.ts',
        output: {
            path: path.resolve(__dirname, 'mailman/assets/static/main/js/'),
            filename: 'app.js'
        },
        ...baseConfig([
            tsLoaderConfig, cssLoaderConfig,
        ]),
    },
    {
        entry: './mailman/chooser/static_src/index.ts',
        output: {
            'path': path.resolve(__dirname, 'mailman/chooser/assets/static/chooser/js/'),
            'filename': 'index.js'
        },
        ...baseConfig([
            tsLoaderConfig, tsxLoaderConfig,
        ]),
    },
]
