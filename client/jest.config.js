/** @type {import('jest').Config} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  roots: ['<rootDir>/src'],
  testMatch: ['**/*.test.ts'],
  moduleFileExtensions: ['ts', 'js'],
  moduleNameMapper: {
    // Mock CSS imports
    '\\.css$': '<rootDir>/src/__mocks__/styleMock.js',
  },
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
    '!src/browser.ts',
  ],
};
