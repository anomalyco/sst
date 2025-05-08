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
exports.siteBuilder = siteBuilder;
var pulumi_1 = require("@pulumi/pulumi");
var semaphore_1 = require("../../../util/semaphore");
var command_1 = require("@pulumi/command");
var limiter = new semaphore_1.Semaphore(parseInt(process.env.SST_BUILD_CONCURRENCY_SITE || "1"));
function siteBuilder(name, args, opts) {
    var _this = this;
    // Wait for the all args values to be resolved before acquiring the semaphore
    return (0, pulumi_1.all)([args]).apply(function (_a) { return __awaiter(_this, [_a], void 0, function (_b) {
        var waitOn, command;
        var args = _b[0];
        return __generator(this, function (_c) {
            switch (_c.label) {
                case 0: return [4 /*yield*/, limiter.acquire(name)];
                case 1:
                    _c.sent();
                    command = new command_1.local.Command(name, args, opts);
                    waitOn = command.urn;
                    // When running `sst diff`, `local.Command`'s `create` and `update` are not called.
                    // So we will also run `local.runOutput` to get the output of the command.
                    if ($cli.command === "diff") {
                        waitOn = command_1.local.runOutput({
                            command: args.create,
                            dir: args.dir,
                            environment: args.environment,
                        }, opts).stdout;
                    }
                    return [2 /*return*/, waitOn.apply(function () {
                            limiter.release();
                            return command;
                        })];
            }
        });
    }); });
}
