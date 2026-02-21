module.exports = [
  {
    ignores: ["node_modules/"],
  },
  {
    rules: {
      "no-unused-vars": "error",
      "no-undef": "error",
    },
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: "commonjs",
      globals: {
        require: "readonly",
        module: "readonly",
        exports: "readonly",
        process: "readonly",
        __dirname: "readonly",
        console: "readonly",
      },
    },
  },
];
