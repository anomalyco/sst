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
exports.KvData = void 0;
var fs_1 = require("fs");
var pulumi_1 = require("@pulumi/pulumi");
var fetch_js_1 = require("../helpers/fetch.js");
var Provider = /** @class */ (function () {
    function Provider() {
    }
    Provider.prototype.create = function (inputs) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.upload(inputs.accountId, inputs.namespaceId, inputs.entries, [])];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, { id: "data" }];
                }
            });
        });
    };
    Provider.prototype.update = function (id, olds, news) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.upload(news.accountId, news.namespaceId, news.entries, news.namespaceId === olds.namespaceId ? olds.entries : [])];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, {}];
                }
            });
        });
    };
    Provider.prototype.upload = function (accountId, namespaceId, entries, oldEntries) {
        return __awaiter(this, void 0, void 0, function () {
            var oldFilesMap;
            var _this = this;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        oldFilesMap = new Map(oldEntries.map(function (f) { return [f.key, f]; }));
                        return [4 /*yield*/, Promise.all(entries
                                .filter(function (entry) {
                                var old = oldFilesMap.get(entry.key);
                                return ((old === null || old === void 0 ? void 0 : old.hash) !== entry.hash ||
                                    (old === null || old === void 0 ? void 0 : old.contentType) !== entry.contentType ||
                                    (old === null || old === void 0 ? void 0 : old.cacheControl) !== entry.cacheControl);
                            })
                                .map(function (entry) { return __awaiter(_this, void 0, void 0, function () {
                                var formData, _a, _b, _c, error_1;
                                return __generator(this, function (_d) {
                                    switch (_d.label) {
                                        case 0:
                                            formData = new FormData();
                                            formData.append("metadata", JSON.stringify({
                                                contentType: entry.contentType,
                                                cacheControl: entry.cacheControl,
                                            }));
                                            //formData.append("value", fs.createReadStream(entry.source));
                                            _b = (_a = formData).append;
                                            _c = ["value"];
                                            return [4 /*yield*/, fs_1.default.promises.readFile(entry.source, "base64")];
                                        case 1:
                                            //formData.append("value", fs.createReadStream(entry.source));
                                            _b.apply(_a, _c.concat([_d.sent()]));
                                            _d.label = 2;
                                        case 2:
                                            _d.trys.push([2, 4, , 5]);
                                            return [4 /*yield*/, (0, fetch_js_1.cfFetch)("/accounts/".concat(accountId, "/storage/kv/namespaces/").concat(namespaceId, "/values/").concat(entry.key), {
                                                    method: "PUT",
                                                    body: formData,
                                                })];
                                        case 3:
                                            _d.sent();
                                            return [3 /*break*/, 5];
                                        case 4:
                                            error_1 = _d.sent();
                                            console.log(error_1);
                                            throw error_1;
                                        case 5: return [2 /*return*/];
                                    }
                                });
                            }); }))];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    return Provider;
}());
var KvData = /** @class */ (function (_super) {
    __extends(KvData, _super);
    function KvData(name, args, opts) {
        return _super.call(this, new Provider(), "".concat(name, ".sst.cloudflare.KvPairs"), args, opts) || this;
    }
    return KvData;
}(pulumi_1.dynamic.Resource));
exports.KvData = KvData;
