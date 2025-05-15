class ChangeProfilesAgeAndNameNullConstraint < ActiveRecord::Migration[6.1]
  def up
    # 既存の NULL レコードをバックフィル
    Profile.where(age: nil).update_all(age: 0)
    Profile.where(name: nil).update_all(name: "")
    # NOT NULL 制約を追加
    change_column_null :profiles, :age,  false
    change_column_null :profiles, :name, false
  end

  def down
    # ロールバック時に制約を外す
    change_column_null :profiles, :age,  true
    change_column_null :profiles, :name, true
  end
end
