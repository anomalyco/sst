// @ts-nocheck

import fs from "fs";
import ts from "typescript";
import prettier from "prettier";

const config = process.argv[2];
const provider = process.argv[3];
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
    'The "providers" property must be a plain object, not a dynamic expression like a ternary or variable.',
  );
  process.exit(1);
}

function getPropertyName(property) {
  return property.name.getText().replace(/^['"]|['"]$/g, "");
}

function createStringProperty(name, value) {
  return ts.factory.createPropertyAssignment(
    name,
    ts.factory.createStringLiteral(value),
  );
}

function createProviderValue(versionValue) {
  if (!pkgName) {
    return ts.factory.createStringLiteral(versionValue);
  }

  return ts.factory.createObjectLiteralExpression(
    [
      createStringProperty("package", pkgName),
      createStringProperty("version", versionValue),
    ],
    false,
  );
}

function upsertObjectProperty(properties, name, initializer, overwrite) {
  const index = properties.findIndex(
    (property) =>
      ts.isPropertyAssignment(property) && getPropertyName(property) === name,
  );

  if (index === -1) {
    properties.push(ts.factory.createPropertyAssignment(name, initializer));
    return;
  }

  if (!overwrite) {
    return;
  }

  properties.splice(
    index,
    1,
    ts.factory.createPropertyAssignment(
      properties[index].name,
      initializer,
    ),
  );
}

function updateProviderValue(initializer) {
  if (!pkgName) {
    return ts.factory.createStringLiteral(version);
  }

  if (ts.isStringLiteralLike(initializer)) {
    return createProviderValue(initializer.text);
  }

  if (!ts.isObjectLiteralExpression(initializer)) {
    return createProviderValue(version);
  }

  const properties = [...initializer.properties];
  upsertObjectProperty(
    properties,
    "package",
    ts.factory.createStringLiteral(pkgName),
    true,
  );
  upsertObjectProperty(
    properties,
    "version",
    ts.factory.createStringLiteral(version),
    false,
  );
  return ts.factory.createObjectLiteralExpression(properties, false);
}

const existingIndex = providersProperty.initializer.properties.findIndex(
  (property) =>
    ts.isPropertyAssignment(property) &&
    getPropertyName(property) === provider,
);

const newProperty = ts.factory.createPropertyAssignment(
  ts.factory.createStringLiteral(provider),
  existingIndex === -1
    ? createProviderValue(version)
    : updateProviderValue(
        providersProperty.initializer.properties[existingIndex].initializer,
      ),
);

if (existingIndex === -1) {
  providersProperty.initializer.properties.push(newProperty);
} else {
  providersProperty.initializer.properties.splice(existingIndex, 1, newProperty);
}

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
