// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  ignore: ['**/.sst/**'],
  compatibilityDate: '2025-05-15',
  nitro: {
    preset: 'aws-lambda',
    awsLambda: {
      streaming: true
    }
  },
  devtools: { enabled: true }
})
