/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

const path = require('path')
const VueLoaderPlugin = require('vue-loader/lib/plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const fs = require('fs')
const webpack = require('webpack')

module.exports = {
    mode: process.env.NODE_ENV !== 'production' ? 'development' : 'production',
    devtool: process.env.NODE_ENV !== 'production' ? 'eval-source-map' : 'cheap-source-map',
    entry: {
        'generic-forms': path.resolve(__dirname, 'src', 'generic-forms.js'),
        'account-delete': path.resolve(__dirname, 'src', 'account-delete.js'),
        'account': path.resolve(__dirname, 'src', 'account.js'),
        'grid-customize': path.resolve(__dirname, 'src', 'grid-customize.js'),
        'grid': path.resolve(__dirname, 'src', 'grid.js'),
    },
    output: {
        filename: '[name].js',
        path: path.resolve(__dirname, 'static', 'dist')
    },
    resolve: {
        alias: {
            '@': path.resolve(__dirname, 'src'),
        },
        extensions: [ '.vue', '.js', '.scss', '.css' ]
    },
    module: {
        rules: [
            {
                test: /\.vue$/,
                loader: 'vue-loader'
            },
            {
                test: /\.js$/,
                loader: 'babel-loader'
            },
            {
                test: /\.s?css$/,
                use: [
                    process.env.NODE_ENV !== 'production' ? 'vue-style-loader' : MiniCssExtractPlugin.loader,
                    'css-loader',
                    'sass-loader'
                ]
            }
        ]
    },
    plugins: [
        new VueLoaderPlugin(),
        new MiniCssExtractPlugin({
            filename: '[name].css'
        }),
        new webpack.BannerPlugin(fs.readFileSync(path.resolve(__dirname, 'license-header.txt'), 'utf8'))
    ]
}