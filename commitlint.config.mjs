export default {
  extends: ['@commitlint/config-conventional'],
  plugins: [
    {
      rules: {
        'no-breaking-change': ({ header, notes }) => {
          const bang = /^\w+(\([^)]*\))?!:/.test(header ?? '');
          const footer = notes.some((n) => /^BREAKING[ -]CHANGE$/.test(n.title));
          return [
            !bang && !footer,
            'breaking changes (major releases) are not allowed: remove "!" from the header and any BREAKING CHANGE footer',
          ];
        },
      },
    },
  ],
  rules: {
    'no-breaking-change': [2, 'always'],
  },
};
