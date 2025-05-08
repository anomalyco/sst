"use strict";
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
exports.findAbove = findAbove;
exports.findBelow = findBelow;
exports.isChild = isChild;
exports.existsAsync = existsAsync;
var promises_1 = require("fs/promises");
var path_1 = require("path");
function findAbove(dir, target) {
    return __awaiter(this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    if (dir === "/")
                        return [2 /*return*/, undefined];
                    return [4 /*yield*/, existsAsync(path_1.default.join(dir, target))];
                case 1:
                    if (_a.sent())
                        return [2 /*return*/, dir];
                    return [2 /*return*/, findAbove(path_1.default.resolve(path_1.default.join(dir, "..")), target)];
            }
        });
    });
}
function findBelow(dir, target) {
    return __awaiter(this, void 0, void 0, function () {
        function loop(dir) {
            return __awaiter(this, void 0, void 0, function () {
                var current, files, _i, files_1, file, full, result;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            current = path_1.default.join(dir, target);
                            return [4 /*yield*/, existsAsync(current)];
                        case 1:
                            if (_a.sent())
                                return [2 /*return*/, dir];
                            return [4 /*yield*/, promises_1.default.readdir(dir, { withFileTypes: true })];
                        case 2:
                            files = _a.sent();
                            _i = 0, files_1 = files;
                            _a.label = 3;
                        case 3:
                            if (!(_i < files_1.length)) return [3 /*break*/, 6];
                            file = files_1[_i];
                            if (file.name === "node_modules")
                                return [3 /*break*/, 5];
                            if (file.name === ".sst")
                                return [3 /*break*/, 5];
                            if (!file.isDirectory()) return [3 /*break*/, 5];
                            full = path_1.default.join(dir, file.name);
                            return [4 /*yield*/, loop(full)];
                        case 4:
                            result = _a.sent();
                            if (result)
                                return [2 /*return*/, result];
                            _a.label = 5;
                        case 5:
                            _i++;
                            return [3 /*break*/, 3];
                        case 6: return [2 /*return*/];
                    }
                });
            });
        }
        var value;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0: return [4 /*yield*/, loop(dir)];
                case 1:
                    value = _a.sent();
                    if (!value)
                        throw new Error("Could not find a ".concat(target, " file"));
                    return [2 /*return*/, value];
            }
        });
    });
}
function isChild(parent, child) {
    var relative = path_1.default.relative(parent, child);
    return Boolean(relative && !relative.startsWith("..") && !path_1.default.isAbsolute(relative));
}
function existsAsync(input) {
    return __awaiter(this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            return [2 /*return*/, promises_1.default
                    .access(input)
                    .then(function () { return true; })
                    .catch(function () { return false; })];
        });
    });
}
