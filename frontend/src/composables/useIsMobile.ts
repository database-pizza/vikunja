import {createSharedComposable, useMediaQuery} from '@vueuse/core'
import {BULMA_MOBILE_BREAKPOINT} from '@/constants/breakpoints'

export const useIsMobile = createSharedComposable(() => useMediaQuery(`(max-width: ${BULMA_MOBILE_BREAKPOINT}px)`))
