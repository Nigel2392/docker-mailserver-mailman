const path = require('path');

module.exports = {
    mode: 'development', 
    entry: './static_src/app.ts', 
    
    output: {
        path: path.resolve(__dirname, 'mailman/assets/static/main/js/'),
        filename: 'app.js'
    },
    
    resolve: {
        extensions: ['.ts', '.js', '.css'],
    },
    
    module: {
        rules: [
            {
                test: /\.ts$/,
                use: 'ts-loader',
                exclude: /node_modules/
            },
            // Add this block for CSS
            {
                test: /\.css$/i,
                use: [
                    'style-loader',
                    'css-loader',
                ],
                exclude: '/node_modules/'
            }
        ]
    }
};