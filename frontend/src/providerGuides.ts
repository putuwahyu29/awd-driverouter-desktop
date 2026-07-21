export const PROVIDER_GUIDES: Record<string, (t: (key: string) => string) => { title: string; steps: string }> = {
  google: (t) => ({ title: t('googleGuideTitle'), steps: t('googleGuideSteps') }),
  onedrive: (t) => ({ title: t('onedriveGuideTitle'), steps: t('onedriveGuideSteps') }),
  dropbox: (t) => ({ title: t('dropboxGuideTitle'), steps: t('dropboxGuideSteps') }),
  box: (t) => ({ title: t('boxGuideTitle'), steps: t('boxGuideSteps') }),
  yandex: (t) => ({ title: t('yandexGuideTitle'), steps: t('yandexGuideSteps') }),
  pcloud: (t) => ({ title: t('pcloudGuideTitle'), steps: t('pcloudGuideSteps') }),
  telegram_user: (t) => ({ title: t('tgUserGuideTitle'), steps: t('tgUserGuideSteps') }),
};
