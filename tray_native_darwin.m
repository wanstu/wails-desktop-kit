//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#include <stdint.h>
#include <string.h>

extern void dkTrayClosed(uintptr_t token);
extern void dkTrayClicked(uintptr_t token, int index);

// All AppKit work is dispatched onto the existing Wails main loop. This
// adapter never starts/stops NSApplication and never changes its Dock policy.
@interface DKTrayTarget : NSObject
@property(nonatomic) uintptr_t token;
@property(nonatomic,strong) NSStatusItem *item;
@property(nonatomic,strong) NSMenu *menu;
@property(nonatomic,strong) NSMutableDictionary<NSNumber*,NSMenuItem*> *entries;
@end

@implementation DKTrayTarget
- (void)activate:(id)sender {
    if (NSApp.currentEvent.type == NSEventTypeRightMouseUp) {
        [self.item popUpStatusItemMenu:self.menu];
    } else {
        dkTrayClicked(self.token, -1);
    }
}
- (void)selectItem:(NSMenuItem*)sender {
    dkTrayClicked(self.token, (int)sender.tag);
}
@end

static NSMutableDictionary<NSNumber*,DKTrayTarget*> *DKTrays;
static NSMutableSet<NSNumber*> *DKClosed;

static void DKOnMain(dispatch_block_t work) {
    if (NSThread.isMainThread) { work(); }
    else { dispatch_sync(dispatch_get_main_queue(), work); }
}
static void DKInit(void) {
    if (!DKTrays) DKTrays = [NSMutableDictionary new];
    if (!DKClosed) DKClosed = [NSMutableSet new];
}
int dkTrayCreate(uintptr_t token, const char *items, const char *title, const void *icon, int length) {
    __block int ok = 0;
    // Inputs are consumed before this synchronous call returns.
    DKOnMain(^{
        @autoreleasepool {
            DKInit();
            NSNumber *key = @(token);
            if ([DKClosed containsObject:key]) return;
            NSData *data = [NSData dataWithBytes:items length:strlen(items)];
            NSArray *specs = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
            NSImage *image = [[NSImage alloc] initWithData:[NSData dataWithBytes:icon length:length]];
            if (!specs || !image) return;
            image.size = NSMakeSize(18,18);
            DKTrayTarget *target = [DKTrayTarget new];
            target.token = token;
            target.item = [NSStatusBar.systemStatusBar statusItemWithLength:NSVariableStatusItemLength];
            if (!target.item || !target.item.button) {
                if (target.item) [NSStatusBar.systemStatusBar removeStatusItem:target.item];
                return;
            }
            target.menu = [NSMenu new];
            target.menu.autoenablesItems = NO;
            target.entries = [NSMutableDictionary new];
            NSInteger index = 0;
            for (NSDictionary *spec in specs) {
                if ([spec[@"Separator"] boolValue]) {
                    [target.menu addItem:NSMenuItem.separatorItem];
                } else {
                    NSMenuItem *entry = [[NSMenuItem alloc] initWithTitle:spec[@"Label"] action:@selector(selectItem:) keyEquivalent:@""];
                    int action = [spec[@"WindowAction"] intValue];
                    entry.tag = action ? action : index;
                    entry.target = target;
                    entry.state = [spec[@"Checked"] boolValue] ? NSControlStateValueOn : NSControlStateValueOff;
                    entry.enabled = ![spec[@"Disabled"] boolValue];
                    target.entries[@(index)] = entry;
                    [target.menu addItem:entry];
                }
                index++;
            }
            target.item.button.image = image;
            target.item.button.toolTip = [NSString stringWithUTF8String:title];
            target.item.button.target = target;
            target.item.button.action = @selector(activate:);
            [target.item.button sendActionOn:NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp];
            DKTrays[key] = target;
            ok = 1;
        }
    });
    return ok;
}
void dkTrayUpdate(uintptr_t token, int index, int checked, int enabled) {
    // Async updates cannot deadlock shutdown. Lookup at execution time makes
    // an update queued before removal harmless after removal.
    dispatch_async(dispatch_get_main_queue(), ^{
        NSMenuItem *entry = DKTrays[@(token)].entries[@(index)];
        entry.state = checked ? NSControlStateValueOn : NSControlStateValueOff;
        entry.enabled = enabled != 0;
    });
}
void dkTrayClose(uintptr_t token) {
    dispatch_async(dispatch_get_main_queue(), ^{
        DKInit();
        [DKClosed addObject:@(token)];
        DKTrayTarget *target = DKTrays[@(token)];
        if (target) {
            target.item.button.target = nil;
            [NSStatusBar.systemStatusBar removeStatusItem:target.item];
            [DKTrays removeObjectForKey:@(token)];
        }
        dkTrayClosed(token);
    });
}
void dkTrayForget(uintptr_t token) {
    // Do not wait on a main loop that Wails may already have stopped.
    dispatch_async(dispatch_get_main_queue(), ^{ [DKClosed removeObject:@(token)]; });
}
