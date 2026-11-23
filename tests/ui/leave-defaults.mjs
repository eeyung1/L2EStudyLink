// Existing journey fixtures deliberately navigate away from drafts. The focused
// list/safety suite separately asserts Stay/Leave behavior and request counts.
export async function launchWithLeaveDefaults(chromium){
 const browser=await chromium.launch(),create=browser.newContext.bind(browser);
 browser.newContext=async options=>{const context=await create(options);context.on('page',page=>page.on('dialog',dialog=>dialog.type()==='confirm'||dialog.type()==='beforeunload'?dialog.accept():dialog.dismiss()));return context;};return browser;
}
