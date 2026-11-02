import { ref } from 'vue'
import { useAppStore } from '../stores/app'

/**
 * 店面模板系统（站长全局切换 · 渐进并行迁移）
 *
 * - 当前激活模板优先级：本地预览覆盖(?template=) > 站点全局配置(storefront_template) > 上次访问时的模板 > 默认 classic
 * - classic 使用 ../views/*；其他主题先加载专属页面，再复用 vault 页面，最后回退 classic。
 */

export type StorefrontTemplate = 'classic' | 'vault' | 'mint' | 'paper' | 'mono'

export const STOREFRONT_TEMPLATES: StorefrontTemplate[] = ['classic', 'vault', 'mint', 'paper', 'mono']
export const DEFAULT_TEMPLATE: StorefrontTemplate = 'classic'

const OVERRIDE_KEY = 'dj-storefront-template'
// 站点配置是异步拿到的；记住上次生效的模板，配置返回前先按它渲染，避免每次打开都先闪一下 classic 外壳
const REMEMBER_KEY = 'dj-storefront-template-last'

const isTemplate = (value: unknown): value is StorefrontTemplate =>
    typeof value === 'string' && (STOREFRONT_TEMPLATES as string[]).includes(value)

const readOverride = (): StorefrontTemplate | null => {
    try {
        const value = localStorage.getItem(OVERRIDE_KEY)
        return isTemplate(value) ? value : null
    } catch {
        return null
    }
}

// 固定当前标签页的预览选择，避免同时比较多个主题时被其他标签页覆盖。
const previewTemplate = ref<StorefrontTemplate | null>(readOverride())

const readRemembered = (): StorefrontTemplate | null => {
    try {
        const value = localStorage.getItem(REMEMBER_KEY)
        return isTemplate(value) ? value : null
    } catch {
        return null
    }
}

let rememberedTemplate = typeof window === 'undefined' ? null : readRemembered()

/** 站点配置返回后调用：记住这次生效的模板，供下次打开时在配置返回前使用。 */
export const rememberTemplate = (value: unknown): void => {
    if (!isTemplate(value) || value === rememberedTemplate) return
    rememberedTemplate = value
    try {
        localStorage.setItem(REMEMBER_KEY, value)
    } catch {
        /* localStorage 不可用时忽略 */
    }
}

/** 站点配置返回之前，是否已经知道该用哪套模板（预览覆盖或上次访问的记录）。 */
export const hasTemplateHint = (): boolean => previewTemplate.value !== null || rememberedTemplate !== null

/**
 * 预览用：URL 带 ?template=vault / ?template=classic 时持久化到 localStorage，
 * ?template=reset 清除覆盖。站长正式切换走站点配置，不依赖此入口。
 * 在 app 挂载前调用一次即可。
 */
export const initTemplateOverride = (): void => {
    if (typeof window === 'undefined') return
    try {
        const param = new URLSearchParams(window.location.search).get('template')
        // 失效的预览链接也清除本地覆盖，恢复跟随站点配置。
        if (param === 'reset' || (param !== null && !isTemplate(param))) {
            previewTemplate.value = null
            localStorage.removeItem(OVERRIDE_KEY)
        } else if (isTemplate(param)) {
            previewTemplate.value = param
            localStorage.setItem(OVERRIDE_KEY, param)
        }
    } catch {
        /* localStorage 不可用时忽略 */
    }
}

/** 当前激活的店面模板。 */
export const getActiveTemplate = (): StorefrontTemplate => {
    const override = previewTemplate.value
    if (override) return override
    try {
        const appStore = useAppStore()
        const fromConfig = appStore.config?.storefront_template
        if (isTemplate(fromConfig)) return fromConfig
    } catch {
        /* pinia 尚未就绪时退回默认 */
    }
    return rememberedTemplate ?? DEFAULT_TEMPLATE
}

// 主题专属页面优先；新主题共享 vault 的商品、订单和账户页面。
const templateViews = import.meta.glob(['./vault/**/*.vue', './mint/**/*.vue', './paper/**/*.vue', './mono/**/*.vue'])

type ViewLoader = () => Promise<unknown>

/**
 * 路由 view 解析器：主题专属页面 → vault 共享页面 → classic loader。
 * 用法：`component: templateView('Home', () => import('../views/Home.vue'))`
 */
export const templateView = (name: string, classicLoader: ViewLoader): ViewLoader => {
    return () => {
        const template = getActiveTemplate()
        if (template !== 'classic') {
            const loader = templateViews[`./${template}/${name}.vue`] || templateViews[`./vault/${name}.vue`]
            if (loader) return loader()
        }
        return classicLoader()
    }
}
