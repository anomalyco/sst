// @ts-nocheck

import fs from "fs";
import ts from "typescript";
import prettier from "prettier";

const config = process.argv[2];
const pkg = process.argv[3];
const version = process.argv[4];
const pkgName = process.argv[5] || "";

const code = fs.readFileSync(config);

const sourceFile = ts.createSourceFile(
  "temp.ts",
  code.toString(),
  ts.ScriptTarget.Latest,
  true,
);

// Find the default export declaration
const exportAssignment = sourceFile.statements.find((statement) =>
  ts.isExportAssignment(statement),
);

// Find the "$config" call expression
const configCallExpression = exportAssignment.expression;

// Find the "app" function declaration inside the "$config" call
const appFunctionDeclaration =
  configCallExpression.arguments[0].properties.find(
    (property) => property.name.getText() === "app",
  );

const appBody = ts.isMethodDeclaration(appFunctionDeclaration)
  ? appFunctionDeclaration.body
  : appFunctionDeclaration.initializer?.body;

// Concise arrow: app: (input) => ({...})
// Block body: app(input) { return {...}; }
const returnedObject = ts.isParenthesizedExpression(appBody)
  ? appBody.expression
  : ts.isObjectLiteralExpression(appBody)
    ? appBody
    : appBody?.statements?.find(
        (s) =>
          ts.isReturnStatement(s) &&
          ts.isObjectLiteralExpression(s.expression),
      )?.expression;

if (!returnedObject || !ts.isObjectLiteralExpression(returnedObject)) {
  console.error(
    'Could not find the returned object in the "app" function. Make sure it returns an object literal.',
  );
  process.exit(1);
}

// Find the "providers" property inside the "app" function
let providersProperty = returnedObject.properties.find(
  (property) =>
    ts.isPropertyAssignment(property) &&
    property.name.getText() === "providers",
);

if (!providersProperty) {
  providersProperty = ts.factory.createPropertyAssignment(
    "providers",
    ts.factory.createObjectLiteralExpression([]),
  );
  returnedObject.properties.push(providersProperty);
}

if (!ts.isObjectLiteralExpression(providersProperty.initializer)) {
  console.error(
    'The "providers" property must be a plain object, not a dynamic expression like a ternary or variable.'
  );
  process.exit(1);
}

if (
  providersProperty.initializer.properties.find(
    (property) => property.name.getText().replaceAll('"', "") === pkg,
  )
) {
  process.exit(0);
}
// Create a new property node
let newValue;
if (pkgName) {
  newValue = ts.factory.createObjectLiteralExpression([
    ts.factory.createPropertyAssignment(
      "package",
      ts.factory.createStringLiteral(pkgName),
    ),
    ts.factory.createPropertyAssignment(
      "version",
      ts.factory.createStringLiteral(version),
    ),
  ], false);
} else {
  newValue = ts.factory.createStringLiteral(version);
}
const newProperty = ts.factory.createPropertyAssignment(
  ts.factory.createStringLiteral(pkg),
  newValue,
);

providersProperty.initializer.properties.push(newProperty);

const printer = ts.createPrinter();
const modifiedCode = printer.printNode(
  ts.EmitHint.Unspecified,
  sourceFile,
  sourceFile,
);

const formattedCode = await prettier.format(modifiedCode, {
  parser: "typescript",
});
fs.writeFileSync(config, formattedCode);
