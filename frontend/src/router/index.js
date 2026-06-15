import { createRouter, createWebHistory } from 'vue-router';
import LandingPage from '../views/landing/LandingPage.vue';
import MerchantLogin from '../views/merchant/MerchantLogin.vue';
import MerchantConsole from '../views/merchant/MerchantConsole.vue';
import AdminLogin from '../views/admin/AdminLogin.vue';
import AdminConsole from '../views/admin/AdminConsole.vue';
const router = createRouter({
    history: createWebHistory(),
    routes: [
        { path: '/', redirect: '/merchant/login' },
        { path: '/landing/:token', component: LandingPage },
        { path: '/merchant/login', component: MerchantLogin },
        { path: '/merchant/console', component: MerchantConsole },
        { path: '/admin/login', component: AdminLogin },
        { path: '/admin/console', component: AdminConsole }
    ]
});
export default router;
