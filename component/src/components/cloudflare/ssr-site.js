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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.prepare = prepare;
exports.createKvStorage = createKvStorage;
exports.createRouter = createRouter;
exports.validatePlan = validatePlan;
var path_1 = require("path");
var fs_1 = require("fs");
var glob_1 = require("glob");
var crypto_1 = require("crypto");
var pulumi_1 = require("@pulumi/pulumi");
var component_js_1 = require("../component.js");
var error_js_1 = require("../error.js");
var base_site_js_1 = require("../base/base-site.js");
var kv_js_1 = require("./kv.js");
var worker_js_1 = require("./worker.js");
var kv_data_js_1 = require("./providers/kv-data.js");
var account_id_js_1 = require("./account-id.js");
function prepare(args) {
    var sitePath = normalizeSitePath();
    return {
        sitePath: sitePath,
    };
    function normalizeSitePath() {
        return (0, pulumi_1.output)(args.path).apply(function (sitePath) {
            if (!sitePath)
                return ".";
            if (!fs_1.default.existsSync(sitePath)) {
                throw new error_js_1.VisibleError("No site found at \"".concat(path_1.default.resolve(sitePath), "\""));
            }
            return sitePath;
        });
    }
}
function createKvStorage(parent, name, args) {
    var _a;
    return new (kv_js_1.Kv.bind.apply(kv_js_1.Kv, __spreadArray([void 0], (0, component_js_1.transform)((_a = args.transform) === null || _a === void 0 ? void 0 : _a.assets, "".concat(name, "Assets"), {}, {
        parent: parent,
        retainOnDelete: false,
    }), false)))();
}
function createRouter(parent, name, args, outputPath, storage, plan) {
    return (0, pulumi_1.all)([outputPath, plan]).apply(function (_a) {
        var outputPath = _a[0], plan = _a[1];
        var assetManifest = generateAssetManifest();
        var kvData = uploadAssets();
        var server = createServerWorker();
        var router = createRouterWorker();
        return { server: server, router: router };
        function generateAssetManifest() {
            var _this = this;
            return (0, pulumi_1.output)(args.assets).apply(function (assets) { return __awaiter(_this, void 0, void 0, function () {
                var versionedFilesTTL, nonVersionedFilesTTL, manifest, _loop_1, _i, _a, copy;
                var _this = this;
                var _b, _c, _d;
                return __generator(this, function (_e) {
                    switch (_e.label) {
                        case 0:
                            versionedFilesTTL = 31536000;
                            nonVersionedFilesTTL = 86400;
                            manifest = [];
                            _loop_1 = function (copy) {
                                var fileOptions, filesProcessed, _loop_2, _f, _g, fileOption;
                                return __generator(this, function (_h) {
                                    switch (_h.label) {
                                        case 0:
                                            fileOptions = __spreadArray(__spreadArray([
                                                // unversioned files
                                                {
                                                    files: "**",
                                                    ignore: copy.versionedSubDir
                                                        ? path_1.default.posix.join(copy.versionedSubDir, "**")
                                                        : undefined,
                                                    cacheControl: (_b = assets === null || assets === void 0 ? void 0 : assets.nonVersionedFilesCacheHeader) !== null && _b !== void 0 ? _b : "public,max-age=0,s-maxage=".concat(nonVersionedFilesTTL, ",stale-while-revalidate=").concat(nonVersionedFilesTTL),
                                                }
                                            ], (copy.versionedSubDir
                                                ? [
                                                    {
                                                        files: path_1.default.posix.join(copy.versionedSubDir, "**"),
                                                        cacheControl: (_c = assets === null || assets === void 0 ? void 0 : assets.versionedFilesCacheHeader) !== null && _c !== void 0 ? _c : "public,max-age=".concat(versionedFilesTTL, ",immutable"),
                                                    },
                                                ]
                                                : []), true), ((_d = assets === null || assets === void 0 ? void 0 : assets.fileOptions) !== null && _d !== void 0 ? _d : []), true);
                                            filesProcessed = [];
                                            _loop_2 = function (fileOption) {
                                                var files, _j, _k, _l;
                                                return __generator(this, function (_m) {
                                                    switch (_m.label) {
                                                        case 0:
                                                            files = (0, glob_1.globSync)(fileOption.files, {
                                                                cwd: path_1.default.resolve(outputPath, copy.from),
                                                                nodir: true,
                                                                dot: true,
                                                                ignore: fileOption.ignore,
                                                            }).filter(function (file) { return !filesProcessed.includes(file); });
                                                            filesProcessed.push.apply(filesProcessed, files);
                                                            _k = (_j = manifest.push).apply;
                                                            _l = [manifest];
                                                            return [4 /*yield*/, Promise.all(files.map(function (file) { return __awaiter(_this, void 0, void 0, function () {
                                                                    var source, content, hash;
                                                                    var _a;
                                                                    return __generator(this, function (_b) {
                                                                        switch (_b.label) {
                                                                            case 0:
                                                                                source = path_1.default.resolve(outputPath, copy.from, file);
                                                                                return [4 /*yield*/, fs_1.default.promises.readFile(source, 'utf-8')];
                                                                            case 1:
                                                                                content = _b.sent();
                                                                                hash = crypto_1.default
                                                                                    .createHash("sha256")
                                                                                    .update(content)
                                                                                    .digest("hex");
                                                                                return [2 /*return*/, {
                                                                                        source: source,
                                                                                        key: path_1.default.posix.join(copy.to, file),
                                                                                        hash: hash,
                                                                                        cacheControl: fileOption.cacheControl,
                                                                                        contentType: (_a = fileOption.contentType) !== null && _a !== void 0 ? _a : (0, base_site_js_1.getContentType)(file, "UTF-8"),
                                                                                    }];
                                                                        }
                                                                    });
                                                                }); }))];
                                                        case 1:
                                                            _k.apply(_j, _l.concat([(_m.sent())]));
                                                            return [2 /*return*/];
                                                    }
                                                });
                                            };
                                            _f = 0, _g = fileOptions.reverse();
                                            _h.label = 1;
                                        case 1:
                                            if (!(_f < _g.length)) return [3 /*break*/, 4];
                                            fileOption = _g[_f];
                                            return [5 /*yield**/, _loop_2(fileOption)];
                                        case 2:
                                            _h.sent();
                                            _h.label = 3;
                                        case 3:
                                            _f++;
                                            return [3 /*break*/, 1];
                                        case 4: return [2 /*return*/];
                                    }
                                });
                            };
                            _i = 0, _a = plan.assets.copy;
                            _e.label = 1;
                        case 1:
                            if (!(_i < _a.length)) return [3 /*break*/, 4];
                            copy = _a[_i];
                            return [5 /*yield**/, _loop_1(copy)];
                        case 2:
                            _e.sent();
                            _e.label = 3;
                        case 3:
                            _i++;
                            return [3 /*break*/, 1];
                        case 4: return [2 /*return*/, manifest];
                    }
                });
            }); });
        }
        function uploadAssets() {
            return new kv_data_js_1.KvData("".concat(name, "AssetFiles"), {
                accountId: account_id_js_1.DEFAULT_ACCOUNT_ID,
                namespaceId: storage.id,
                entries: assetManifest.apply(function (manifest) {
                    return manifest.map(function (m) { return ({
                        source: m.source,
                        key: m.key,
                        hash: m.hash,
                        cacheControl: m.cacheControl,
                        contentType: m.contentType,
                    }); });
                }),
            }, { parent: parent });
        }
        function createServerWorker() {
            return new worker_js_1.Worker("".concat(name, "Server"), __assign(__assign({}, plan.server), { environment: (0, pulumi_1.output)(args.environment).apply(function (environment) { return (__assign(__assign({}, environment), plan.server.environment)); }), link: (0, pulumi_1.output)(args.link).apply(function (link) {
                    var _a;
                    return __spreadArray(__spreadArray([], ((_a = plan.server.link) !== null && _a !== void 0 ? _a : []), true), (link !== null && link !== void 0 ? link : []), true);
                }), dev: false }), { parent: parent });
        }
        function createRouterWorker() {
            return new worker_js_1.Worker("".concat(name, "Router"), {
                handler: path_1.default.join($cli.paths.platform, "functions", "cf-ssr-site-router-worker"),
                url: true,
                dev: false,
                domain: args.domain,
                build: {
                    esbuild: assetManifest.apply(function (assetManifest) { return ({
                        define: {
                            SST_ASSET_MANIFEST: JSON.stringify(Object.fromEntries(assetManifest.map(function (e) { return [e.key, e.hash]; }))),
                            SST_ROUTES: JSON.stringify(plan.routes),
                        },
                    }); }),
                },
                transform: {
                    worker: function (workerArgs) {
                        workerArgs.kvNamespaceBindings = [
                            {
                                name: "ASSETS",
                                namespaceId: storage.id,
                            },
                        ];
                        workerArgs.serviceBindings = [
                            {
                                name: "SERVER",
                                service: server.nodes.worker.name,
                            },
                        ];
                    },
                },
            }, 
            // create distribution after assets are uploaded
            { dependsOn: kvData, parent: parent });
        }
    });
}
function validatePlan(input) {
    return input;
}
