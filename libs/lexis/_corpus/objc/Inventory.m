// Inventory.m - a small stock list, kept in memory and saved as a property list.
#import <Foundation/Foundation.h>
#import "Inventory.h"

#define INVENTORY_VERSION 2

static NSString *const InventoryDidChange = @"InventoryDidChange";
static const NSUInteger kMaxItems = 0x400;

typedef NS_ENUM(NSInteger, StockLevel) {
    StockLevelEmpty = 0,
    StockLevelLow,
    StockLevelFine,
};

@protocol InventoryObserver <NSObject>
- (void)inventory:(Inventory *)inventory didChangeItem:(NSString *)name;
@optional
- (void)inventoryDidEmpty:(Inventory *)inventory;
@end

@interface Item : NSObject <NSCopying>
@property (nonatomic, copy) NSString *name;
@property (nonatomic, assign) NSInteger count;
@property (nonatomic, readonly) StockLevel level;
+ (instancetype)itemWithName:(NSString *)name count:(NSInteger)count;
@end

@implementation Item

+ (instancetype)itemWithName:(NSString *)name count:(NSInteger)count {
    Item *item = [[self alloc] init];
    item.name = name;
    item.count = count;
    return item;
}

- (StockLevel)level {
    if (self.count <= 0) {
        return StockLevelEmpty;
    }
    return self.count < 5 ? StockLevelLow : StockLevelFine;
}

- (id)copyWithZone:(NSZone *)zone {
    return [Item itemWithName:self.name count:self.count];
}

- (NSString *)description {
    return [NSString stringWithFormat:@"%@ x%ld\n", self.name, (long)self.count];
}

@end

@interface Inventory ()
@property (nonatomic, strong) NSMutableDictionary<NSString *, Item *> *items;
@property (nonatomic, weak) id<InventoryObserver> observer;
@end

@implementation Inventory {
    BOOL _dirty;
}

- (instancetype)init {
    if ((self = [super init])) {
        _items = [NSMutableDictionary dictionary];
        _dirty = NO;
    }
    return self;
}

/* Adding to an item that is not there makes it. */
- (BOOL)add:(NSInteger)count of:(NSString *)name {
    if (name == nil || self.items.count >= kMaxItems) {
        return NO;
    }
    Item *item = self.items[name] ?: [Item itemWithName:name count:0];
    item.count += count;
    self.items[name] = item;
    _dirty = YES;
    [self.observer inventory:self didChangeItem:name];
    [[NSNotificationCenter defaultCenter] postNotificationName:InventoryDidChange object:self];
    return YES;
}

- (NSArray<NSString *> *)namesBelow:(StockLevel)level {
    NSMutableArray *names = [NSMutableArray array];
    for (Item *item in self.items.allValues) {
        if (item.level < level) {
            [names addObject:item.name];
        }
    }
    return [names sortedArrayUsingSelector:@selector(compare:)];
}

- (void)eachItem:(void (^)(Item *item, BOOL *stop))block {
    __block BOOL stop = NO;
    [self.items enumerateKeysAndObjectsUsingBlock:^(NSString *key, Item *item, BOOL *inner) {
        block(item, &stop);
        *inner = stop;
    }];
}

- (BOOL)saveTo:(NSString *)path error:(NSError **)error {
    NSDictionary *plist = @{
        @"version" : @INVENTORY_VERSION,
        @"ratio" : @0.75,
        @"items" : [self.items.allKeys sortedArrayUsingSelector:@selector(compare:)],
        @"flags" : @[ @YES, @NO ],
    };
    @try {
        NSData *data = [NSPropertyListSerialization dataWithPropertyList:plist
                                                                  format:NSPropertyListXMLFormat_v1_0
                                                                 options:0
                                                                   error:error];
        return [data writeToFile:path atomically:YES];
    } @catch (NSException *e) {
        NSLog(@"%@: %@", e.name, e.reason);
        return NO;
    } @finally {
        _dirty = NO;
    }
}

@end
