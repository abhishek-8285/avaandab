const path = require('path');
const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

// Configure Metro to resolve export maps properly for sub-path packages like @posthog/core
config.resolver.unstable_enablePackageExports = true;

module.exports = config;
