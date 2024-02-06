const { utils } = require('@commitlint/config-nx-scopes');

module.exports = {
  extends: ['@commitlint/config-conventional', '@commitlint/config-nx-scopes'],
  rules: {
    'scope-empty': [2, 'never'],
    'scope-enum': async (ctx) => [
      2,
      'always',
      ['ci', 'docs', 'specs', ...(await utils.getProjects(ctx))],
    ],
  },
};