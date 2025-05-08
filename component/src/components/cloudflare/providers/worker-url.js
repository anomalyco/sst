"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.WorkerUrl = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var fetch_js_1 = require("../helpers/fetch.js");
var Provider = /** @class */ (function () {
    function Provider() {
    }
    Provider.prototype.create = function (inputs) {
        return __awaiter(this, void 0, void 0, function () {
            var url;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.process(inputs)];
                    case 1:
                        url = _a.sent();
                        return [2 /*return*/, {
                                id: inputs.scriptName,
                                outs: url ? { url: url } : {},
                            }];
                }
            });
        });
    };
    Provider.prototype.update = function (id, olds, news) {
        return __awaiter(this, void 0, void 0, function () {
            var url;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.process(news)];
                    case 1:
                        url = _a.sent();
                        return [2 /*return*/, {
                                outs: url ? { url: url } : {},
                            }];
                }
            });
        });
    };
    Provider.prototype.process = function (inputs) {
        return __awaiter(this, void 0, void 0, function () {
            var userSubdomain;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!(inputs.enabled === false)) return [3 /*break*/, 2];
                        return [4 /*yield*/, this.setEnabledFlag(inputs)];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, undefined];
                    case 2: return [4 /*yield*/, Promise.all([
                            this.getWorkerDevSubdomain(inputs),
                            this.setEnabledFlag(inputs),
                        ])];
                    case 3:
                        userSubdomain = (_a.sent())[0];
                        return [2 /*return*/, "".concat(inputs.scriptName, ".").concat(userSubdomain, ".workers.dev")];
                }
            });
        });
    };
    Provider.prototype.getWorkerDevSubdomain = function (inputs) {
        return __awaiter(this, void 0, void 0, function () {
            var ret, error_1;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        return [4 /*yield*/, (0, fetch_js_1.cfFetch)("/accounts/".concat(inputs.accountId, "/workers/subdomain"), {
                                headers: { "Content-Type": "application/json" },
                            })];
                    case 1:
                        ret = _a.sent();
                        return [2 /*return*/, ret.result.subdomain];
                    case 2:
                        error_1 = _a.sent();
                        console.log(error_1);
                        throw error_1;
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    Provider.prototype.setEnabledFlag = function (inputs) {
        return __awaiter(this, void 0, void 0, function () {
            var error_2;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 3, , 4]);
                        return [4 /*yield*/, (0, fetch_js_1.cfFetch)("/accounts/".concat(inputs.accountId, "/workers/scripts/").concat(inputs.scriptName, "/subdomain"), {
                                method: "POST",
                                headers: { "Content-Type": "application/json" },
                                body: JSON.stringify({ enabled: inputs.enabled }),
                            })];
                    case 1:
                        _a.sent();
                        // Add a delay when the subdomain is first created.
                        // This is to prevent an issue where a negative cache-hit
                        // causes the subdomain to be unavailable for 30 seconds.
                        // This is a temporary measure until we fix this on the edge.
                        return [4 /*yield*/, new Promise(function (resolve) { return setTimeout(resolve, 3000); })];
                    case 2:
                        // Add a delay when the subdomain is first created.
                        // This is to prevent an issue where a negative cache-hit
                        // causes the subdomain to be unavailable for 30 seconds.
                        // This is a temporary measure until we fix this on the edge.
                        _a.sent();
                        return [3 /*break*/, 4];
                    case 3:
                        error_2 = _a.sent();
                        console.log(error_2);
                        throw error_2;
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    return Provider;
}());
var WorkerUrl = /** @class */ (function (_super) {
    __extends(WorkerUrl, _super);
    function WorkerUrl(name, args, opts) {
        return _super.call(this, new Provider(), "".concat(name, ".sst.cloudflare.WorkerUrl"), __assign(__assign({}, args), { url: undefined }), opts) || this;
    }
    return WorkerUrl;
}(pulumi_1.dynamic.Resource));
exports.WorkerUrl = WorkerUrl;
