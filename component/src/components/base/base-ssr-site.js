"use strict";
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.buildApp = buildApp;
var path_1 = require("path");
var fs_1 = require("fs");
var pulumi_1 = require("@pulumi/pulumi");
var link_js_1 = require("../link.js");
var error_js_1 = require("../error.js");
var site_builder_1 = require("../aws/helpers/site-builder");
function buildApp(parent, name, args, sitePath, buildCommand) {
    return (0, pulumi_1.all)([
        sitePath,
        buildCommand !== null && buildCommand !== void 0 ? buildCommand : args.buildCommand,
        args.link,
        args.environment,
    ]).apply(function (_a) {
        var sitePath = _a[0], userCommand = _a[1], links = _a[2], environment = _a[3];
        var cmd = resolveBuildCommand();
        var result = runBuild();
        return result.id.apply(function () { return sitePath; });
        function resolveBuildCommand() {
            if (userCommand)
                return userCommand;
            // Ensure that the site has a build script defined
            if (!userCommand) {
                if (!fs_1.default.existsSync(path_1.default.join(sitePath, "package.json"))) {
                    throw new error_js_1.VisibleError("No package.json found at \"".concat(sitePath, "\"."));
                }
                var packageJson = JSON.parse(fs_1.default.readFileSync(path_1.default.join(sitePath, "package.json")).toString());
                if (!packageJson.scripts || !packageJson.scripts.build) {
                    throw new error_js_1.VisibleError("No \"build\" script found within package.json in \"".concat(sitePath, "\"."));
                }
            }
            if (fs_1.default.existsSync(path_1.default.join(sitePath, "yarn.lock")) ||
                fs_1.default.existsSync(path_1.default.join($cli.paths.root, "yarn.lock")))
                return "yarn run build";
            if (fs_1.default.existsSync(path_1.default.join(sitePath, "pnpm-lock.yaml")) ||
                fs_1.default.existsSync(path_1.default.join($cli.paths.root, "pnpm-lock.yaml")))
                return "pnpm run build";
            if (fs_1.default.existsSync(path_1.default.join(sitePath, "bun.lockb")) ||
                fs_1.default.existsSync(path_1.default.join($cli.paths.root, "bun.lockb")) ||
                fs_1.default.existsSync(path_1.default.join(sitePath, "bun.lock")) ||
                fs_1.default.existsSync(path_1.default.join($cli.paths.root, "bun.lock")))
                return "bun run build";
            return "npm run build";
        }
        function runBuild() {
            // Build link environment variables to inject
            var linkData = link_js_1.Link.build(links || []);
            var linkEnvs = (0, pulumi_1.output)(linkData).apply(function (linkData) {
                var envs = {
                    SST_RESOURCE_App: JSON.stringify({
                        name: $app.name,
                        stage: $app.stage,
                    }),
                };
                for (var _i = 0, linkData_1 = linkData; _i < linkData_1.length; _i++) {
                    var datum = linkData_1[_i];
                    envs["SST_RESOURCE_".concat(datum.name)] = JSON.stringify(datum.properties);
                }
                return envs;
            });
            // Run build
            return (0, site_builder_1.siteBuilder)("".concat(name, "Builder"), {
                create: cmd,
                update: cmd,
                dir: path_1.default.join($cli.paths.root, sitePath),
                environment: linkEnvs.apply(function (linkEnvs) { return (__assign(__assign(__assign({ SST: "1" }, process.env), environment), linkEnvs)); }),
                triggers: [Date.now().toString()],
            }, {
                parent: parent,
                ignoreChanges: process.env.SKIP ? ["*"] : undefined,
            });
        }
    });
}
