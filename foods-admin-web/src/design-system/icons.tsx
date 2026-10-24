import {
  IconAlertTriangle, IconArmchair, IconArrowBackUp, IconArrowsExchange, IconBan, IconBell, IconBike,
  IconBowlSpoon, IconBox, IconBuilding, IconBuildingStore, IconBulb, IconCalendar, IconCash, IconCashRegister, IconChartBar,
  IconChartPie, IconCheck, IconChefHat, IconChevronLeft, IconChevronRight, IconCircleArrowRight, IconCircleCheck,
  IconCircleMinus, IconCircleX, IconClipboardCheck, IconClipboardList, IconClock, IconContract, IconCreditCard, IconCup,
  IconDeviceDesktop, IconDeviceFloppy, IconDeviceMobile, IconDownload, IconEye, IconFileInvoice, IconFilter, IconGauge,
  IconHeart, IconIceCream2, IconId, IconInfoCircle, IconKey, IconLayout, IconLayoutBoardSplit, IconLayoutGrid,
  IconLayoutSidebarLeftCollapse, IconLayoutSidebarLeftExpand, IconLink, IconLock, IconLogin, IconLogout, IconMail,
  IconMapPin, IconMenu2, IconMessageCircle, IconMinus, IconNote, IconNotebook, IconPackages, IconPencil, IconPercentage,
  IconPhone, IconPlug, IconPlus, IconPower, IconQrcode, IconReceipt, IconRefresh, IconRoute, IconSalad, IconSearch,
  IconSend, IconSettings, IconShare, IconShoppingBag, IconShoppingCart, IconSoup, IconStack2, IconStar, IconTag,
  IconToggleLeftFilled, IconToggleRightFilled, IconToolsKitchen2, IconTrash, IconTruck, IconUpload, IconUser, IconUserCheck,
  IconUsers, IconWallet, IconX, IconArrowUpRight, IconFlame, IconChecks, IconMotorbike, type Icon as TablerIcon,
} from "@tabler/icons-react";

export type IconName = "info"|"grid"|"sales"|"receipt"|"menu"|"utensils"|"chefHat"|"cookingPot"|"kitchen"|"combo"|"stock"|"truck"|"users"|"store"|"settings"|"search"|"bell"|"plus"|"minus"|"chevron"|"chevronLeft"|"filter"|"download"|"upload"|"alert"|"check"|"clock"|"box"|"logout"|"eye"|"lock"|"mail"|"edit"|"power"|"toggleOn"|"toggleOff"|"circleCheck"|"circleMinus"|"circleSlash"|"save"|"refresh"|"close"|"panelCollapse"|"panelExpand"|"qr"|"share"|"trash"|"arrowRightCircle"|"payment"|"cancel"|"availability"|"ledger"|"building"|"contract"|"mapPin"|"userCheck"|"register"|"orders"|"tables"|"layout"|"cash"|"calendar"|"phone"|"kiosk"|"gauge"|"cart"|"route"|"link"|"bike"|"user"|"heart"|"star"|"badge"|"tag"|"wallet"|"pie"|"phoneApp"|"percent"|"layers"|"plug"|"chat"|"login"|"key"|"lightbulb"|"undo"|"transfer"|"invoice"|"dish"|"note"|"send"|"shoppingBag"|"salad"|"dessert"|"drink"|"arrowUpRight"|"flame"|"checks"|"motorbike";
/** Tamaños homologados: 14 junto a texto de 12–13 px, 16 en chips y pestañas, 18 en acciones, 20 en navegación, 24 en estados vacíos. */
export type IconSize=14|16|18|20|24|28|32|36|40|48;

/**
 * Set único del sistema, dibujado con Tabler Icons (MIT): retícula de 24 px,
 * trazo de 1.75 y extremos redondeados en todos los glifos. Los iconos son
 * monocromos y heredan `currentColor`; el color de identidad lo aporta
 * `IconTile`, nunca el propio trazo. Las pantallas usan solo los nombres de
 * `IconName`, de modo que cambiar un glifo se hace únicamente aquí.
 */
const glyphs: Record<IconName, TablerIcon> = {
  info:IconInfoCircle, grid:IconLayoutGrid, sales:IconChartBar, receipt:IconReceipt, menu:IconMenu2,
  utensils:IconToolsKitchen2, chefHat:IconChefHat, cookingPot:IconSoup, kitchen:IconSoup, combo:IconLayoutBoardSplit,
  stock:IconPackages, truck:IconTruck, users:IconUsers, store:IconBuildingStore, settings:IconSettings, search:IconSearch,
  bell:IconBell, plus:IconPlus, minus:IconMinus, chevron:IconChevronRight, chevronLeft:IconChevronLeft, filter:IconFilter,
  download:IconDownload, upload:IconUpload, alert:IconAlertTriangle, check:IconCheck, clock:IconClock, box:IconBox,
  logout:IconLogout, eye:IconEye, lock:IconLock, mail:IconMail, edit:IconPencil, power:IconPower,
  toggleOn:IconToggleRightFilled, toggleOff:IconToggleLeftFilled, circleCheck:IconCircleCheck, circleMinus:IconCircleMinus,
  circleSlash:IconBan, save:IconDeviceFloppy, refresh:IconRefresh, close:IconX, panelCollapse:IconLayoutSidebarLeftCollapse,
  panelExpand:IconLayoutSidebarLeftExpand, qr:IconQrcode, share:IconShare, trash:IconTrash,
  arrowRightCircle:IconCircleArrowRight, payment:IconCreditCard, cancel:IconCircleX, availability:IconClipboardCheck,
  ledger:IconNotebook, building:IconBuilding, contract:IconContract, mapPin:IconMapPin, userCheck:IconUserCheck,
  register:IconCashRegister, orders:IconClipboardList, tables:IconArmchair, layout:IconLayout, cash:IconCash,
  calendar:IconCalendar, phone:IconPhone, kiosk:IconDeviceDesktop, gauge:IconGauge, cart:IconShoppingCart, route:IconRoute,
  link:IconLink, bike:IconBike, user:IconUser, heart:IconHeart, star:IconStar, badge:IconId, tag:IconTag, wallet:IconWallet,
  pie:IconChartPie, phoneApp:IconDeviceMobile, percent:IconPercentage, layers:IconStack2, plug:IconPlug,
  chat:IconMessageCircle, login:IconLogin, key:IconKey, lightbulb:IconBulb, undo:IconArrowBackUp,
  transfer:IconArrowsExchange, invoice:IconFileInvoice, dish:IconBowlSpoon, note:IconNote, send:IconSend,
  shoppingBag:IconShoppingBag, salad:IconSalad, dessert:IconIceCream2, drink:IconCup,
  arrowUpRight:IconArrowUpRight, flame:IconFlame, checks:IconChecks, motorbike:IconMotorbike,
};

export function Icon({name,size=20,className}:{name:IconName,size?:IconSize;className?:string}) {
  const Glyph=glyphs[name];
  return <Glyph size={size} stroke={1.75} aria-hidden="true" className={className}/>;
}
