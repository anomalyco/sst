"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Hint = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var Hint;
(function (Hint) {
    var hints = {};
    function reset() {
        hints = {};
    }
    Hint.reset = reset;
    function register(name, hint) {
        (0, pulumi_1.all)([name]).apply(function (_a) {
            var name = _a[0];
            hints[name] = hint;
        });
    }
    Hint.register = register;
    function list() {
        return hints;
    }
    Hint.list = list;
})(Hint || (exports.Hint = Hint = {}));
