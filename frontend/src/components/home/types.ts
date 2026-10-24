/** 首屏轮播单组文案的结构，字段与 i18n 的 home.hero.slides 一一对应 */
export interface HeroSlideCard {
  kicker: string
  title: string
  desc: string
}

export interface HeroSlide {
  prefix: string
  verb: string
  subtitle: string
  left: HeroSlideCard
  right: HeroSlideCard
}
