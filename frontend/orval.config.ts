import { defineConfig } from 'orval';

export default defineConfig({
  firemail: {
    input: {
      target: '../openapi/firemail.yaml',
    },
    output: {
      mode: 'single',
      target: './src/api/generated/firemail.ts',
      schemas: './src/api/generated/model',
      client: 'fetch',
      clean: true,
    },
  },
});
