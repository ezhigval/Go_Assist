// Docusaurus configuration for Go Assist documentation
// @ts-check

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Go Assist Documentation',
  tagline: 'AI-Driven Execution Platform',
  url: 'https://ezhigval.github.io',
  baseUrl: '/Go_Assist/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'img/favicon.ico',
  organizationName: 'ezhigval',
  projectName: 'Go_Assist',

  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'ru', 'zh'],
    localeConfigs: {
      en: {
        label: 'English',
        direction: 'ltr',
      },
      ru: {
        label: 'Russian',
        direction: 'ltr',
      },
      zh: {
        label: 'Chinese',
        direction: 'ltr',
      },
    },
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve('./navigation/sidebars.json'),
          editUrl: 'https://github.com/ezhigval/Go_Assist/tree/main/docs/',
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig: /** @type {import('@docusaurus/preset-classic').ThemeConfig} */ ({
    navbar: {
      title: 'Go Assist',
      logo: {
        alt: 'Go Assist Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'mainSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          to: 'https://github.com/ezhigval/Go_Assist',
          label: 'GitHub',
          position: 'right',
        },
        {
          type: 'localeDropdown',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {
              label: 'Architecture',
              to: '/docs/i18n/en/architecture/README',
            },
            {
              label: 'Concepts',
              to: '/docs/i18n/en/concepts/README',
            },
            {
              label: 'Modules',
              to: '/docs/i18n/en/modules/README',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/ezhigval/Go_Assist',
            },
            {
              label: 'Issues',
              href: 'https://github.com/ezhigval/Go_Assist/issues',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'MIT License',
              href: 'https://github.com/ezhigval/Go_Assist/blob/main/LICENSE',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Go Assist Contributors. Built with Docusaurus.`,
    },
    prism: {
      theme: lightCodeTheme,
      darkTheme: darkCodeTheme,
      additionalLanguages: ['go', 'yaml', 'json', 'bash'],
    },
    mermaid: {
      theme: {
        light: 'default',
        dark: 'dark',
      },
    },
  }),

  plugins: [
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'architecture',
        path: 'i18n',
        include: ['ru/architecture', 'en/architecture', 'zh/architecture'],
        routeBasePath: 'architecture',
        sidebarPath: require.resolve('./navigation/sidebars.json'),
      },
    ],
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'concepts',
        path: 'i18n',
        include: ['ru/concepts', 'en/concepts', 'zh/concepts'],
        routeBasePath: 'concepts',
        sidebarPath: require.resolve('./navigation/sidebars.json'),
      },
    ],
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'modules',
        path: 'i18n',
        include: ['ru/modules', 'en/modules', 'zh/modules'],
        routeBasePath: 'modules',
        sidebarPath: require.resolve('./navigation/sidebars.json'),
      },
    ],
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'ai',
        path: 'i18n',
        include: ['ru/ai', 'en/ai', 'zh/ai'],
        routeBasePath: 'ai',
        sidebarPath: require.resolve('./navigation/sidebars.json'),
      },
    ],
    [
      '@docusaurus/plugin-mermaid',
      {
        id: 'mermaid',
      },
    ],
  ],

  themes: [
    [
      '@docusaurus/theme-mermaid',
      {
        id: 'mermaid-theme',
      },
    ],
  ],

  webpack: {
    jsLoader: (isServer) => ({
      loader: require.resolve('swc-loader'),
      options: {
        jsc: {
          target: 'es2017',
          parser: {
            syntax: 'typescript',
            tsx: true,
          },
          transform: {
            react: {
              runtime: 'automatic',
            },
          },
        },
        module: {
          type: isServer ? 'commonjs' : 'es6',
        },
      },
    }),
  },
};

module.exports = config;
