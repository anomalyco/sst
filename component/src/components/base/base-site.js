"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getContentType = getContentType;
var path_1 = require("path");
function getContentType(filename, textEncoding) {
    var _a;
    var _b;
    var ext = filename.endsWith(".well-known/site-association-json")
        ? ".json"
        : path_1.default.extname(filename);
    var extensions = (_a = {},
        _a[".txt"] = { mime: "text/plain", isText: true },
        _a[".htm"] = { mime: "text/html", isText: true },
        _a[".html"] = { mime: "text/html", isText: true },
        _a[".xhtml"] = { mime: "application/xhtml+xml", isText: true },
        _a[".css"] = { mime: "text/css", isText: true },
        _a[".js"] = { mime: "text/javascript", isText: true },
        _a[".mjs"] = { mime: "text/javascript", isText: true },
        _a[".apng"] = { mime: "image/apng", isText: false },
        _a[".avif"] = { mime: "image/avif", isText: false },
        _a[".gif"] = { mime: "image/gif", isText: false },
        _a[".jpeg"] = { mime: "image/jpeg", isText: false },
        _a[".jpg"] = { mime: "image/jpeg", isText: false },
        _a[".png"] = { mime: "image/png", isText: false },
        _a[".svg"] = { mime: "image/svg+xml", isText: true },
        _a[".bmp"] = { mime: "image/bmp", isText: false },
        _a[".tiff"] = { mime: "image/tiff", isText: false },
        _a[".webp"] = { mime: "image/webp", isText: false },
        _a[".ico"] = { mime: "image/vnd.microsoft.icon", isText: false },
        _a[".eot"] = { mime: "application/vnd.ms-fontobject", isText: false },
        _a[".ttf"] = { mime: "font/ttf", isText: false },
        _a[".otf"] = { mime: "font/otf", isText: false },
        _a[".woff"] = { mime: "font/woff", isText: false },
        _a[".woff2"] = { mime: "font/woff2", isText: false },
        _a[".json"] = { mime: "application/json", isText: true },
        _a[".jsonld"] = { mime: "application/ld+json", isText: true },
        _a[".xml"] = { mime: "application/xml", isText: true },
        _a[".pdf"] = { mime: "application/pdf", isText: false },
        _a[".zip"] = { mime: "application/zip", isText: false },
        _a[".wasm"] = { mime: "application/wasm", isText: false },
        _a[".webmanifest"] = { mime: "application/manifest+json", isText: true },
        _a);
    var extensionData = extensions[ext];
    var mime = (_b = extensionData === null || extensionData === void 0 ? void 0 : extensionData.mime) !== null && _b !== void 0 ? _b : "application/octet-stream";
    var charset = (extensionData === null || extensionData === void 0 ? void 0 : extensionData.isText) && textEncoding !== "none"
        ? ";charset=".concat(textEncoding)
        : "";
    return "".concat(mime).concat(charset);
}
